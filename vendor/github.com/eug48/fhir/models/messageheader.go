// Copyright (c) 2011-2017, HL7, Inc & The MITRE Corporation
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
//     * Redistributions of source code must retain the above copyright notice, this
//       list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above copyright notice,
//       this list of conditions and the following disclaimer in the documentation
//       and/or other materials provided with the distribution.
//     * Neither the name of HL7 nor the names of its contributors may be used to
//       endorse or promote products derived from this software without specific
//       prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED.
// IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT,
// INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT
// NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
// PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY,
// WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package models

import (
	"encoding/json"
	"errors"
	"fmt"
)

type MessageHeader struct {
	DomainResource `bson:",inline"`
	Event          *Coding                                    `bson:"event,omitempty" json:"event,omitempty"`
	Destination    []MessageHeaderMessageDestinationComponent `bson:"destination,omitempty" json:"destination,omitempty"`
	Receiver       *Reference                                 `bson:"receiver,omitempty" json:"receiver,omitempty"`
	Sender         *Reference                                 `bson:"sender,omitempty" json:"sender,omitempty"`
	Timestamp      *FHIRDateTime                              `bson:"timestamp,omitempty" json:"timestamp,omitempty"`
	Enterer        *Reference                                 `bson:"enterer,omitempty" json:"enterer,omitempty"`
	Author         *Reference                                 `bson:"author,omitempty" json:"author,omitempty"`
	Source         *MessageHeaderMessageSourceComponent       `bson:"source,omitempty" json:"source,omitempty"`
	Responsible    *Reference                                 `bson:"responsible,omitempty" json:"responsible,omitempty"`
	Reason         *CodeableConcept                           `bson:"reason,omitempty" json:"reason,omitempty"`
	Response       *MessageHeaderResponseComponent            `bson:"response,omitempty" json:"response,omitempty"`
	Focus          []Reference                                `bson:"focus,omitempty" json:"focus,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *MessageHeader) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "MessageHeader"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to MessageHeader), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *MessageHeader) GetBSON() (interface{}, error) {
	x.ResourceType = "MessageHeader"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "messageHeader" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type messageHeader MessageHeader

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *MessageHeader) UnmarshalJSON(data []byte) (err error) {
	x2 := messageHeader{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = MessageHeader(x2)
		return x.checkResourceType()
	}
	return
}

func (x *MessageHeader) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "MessageHeader"
	} else if x.ResourceType != "MessageHeader" {
		return errors.New(fmt.Sprintf("Expected resourceType to be MessageHeader, instead received %s", x.ResourceType))
	}
	return nil
}

type MessageHeaderMessageDestinationComponent struct {
	BackboneElement `bson:",inline"`
	Name            string     `bson:"name,omitempty" json:"name,omitempty"`
	Target          *Reference `bson:"target,omitempty" json:"target,omitempty"`
	Endpoint        string     `bson:"endpoint,omitempty" json:"endpoint,omitempty"`
}

type MessageHeaderMessageSourceComponent struct {
	BackboneElement `bson:",inline"`
	Name            string        `bson:"name,omitempty" json:"name,omitempty"`
	Software        string        `bson:"software,omitempty" json:"software,omitempty"`
	Version         string        `bson:"version,omitempty" json:"version,omitempty"`
	Contact         *ContactPoint `bson:"contact,omitempty" json:"contact,omitempty"`
	Endpoint        string        `bson:"endpoint,omitempty" json:"endpoint,omitempty"`
}

type MessageHeaderResponseComponent struct {
	BackboneElement `bson:",inline"`
	Identifier      string     `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Code            string     `bson:"code,omitempty" json:"code,omitempty"`
	Details         *Reference `bson:"details,omitempty" json:"details,omitempty"`
}
