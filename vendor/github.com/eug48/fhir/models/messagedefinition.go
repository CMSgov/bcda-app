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

type MessageDefinition struct {
	DomainResource   `bson:",inline"`
	Url              string                                      `bson:"url,omitempty" json:"url,omitempty"`
	Identifier       *Identifier                                 `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Version          string                                      `bson:"version,omitempty" json:"version,omitempty"`
	Name             string                                      `bson:"name,omitempty" json:"name,omitempty"`
	Title            string                                      `bson:"title,omitempty" json:"title,omitempty"`
	Status           string                                      `bson:"status,omitempty" json:"status,omitempty"`
	Experimental     *bool                                       `bson:"experimental,omitempty" json:"experimental,omitempty"`
	Date             *FHIRDateTime                               `bson:"date,omitempty" json:"date,omitempty"`
	Publisher        string                                      `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Contact          []ContactDetail                             `bson:"contact,omitempty" json:"contact,omitempty"`
	Description      string                                      `bson:"description,omitempty" json:"description,omitempty"`
	UseContext       []UsageContext                              `bson:"useContext,omitempty" json:"useContext,omitempty"`
	Jurisdiction     []CodeableConcept                           `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	Purpose          string                                      `bson:"purpose,omitempty" json:"purpose,omitempty"`
	Copyright        string                                      `bson:"copyright,omitempty" json:"copyright,omitempty"`
	Base             *Reference                                  `bson:"base,omitempty" json:"base,omitempty"`
	Parent           []Reference                                 `bson:"parent,omitempty" json:"parent,omitempty"`
	Replaces         []Reference                                 `bson:"replaces,omitempty" json:"replaces,omitempty"`
	Event            *Coding                                     `bson:"event,omitempty" json:"event,omitempty"`
	Category         string                                      `bson:"category,omitempty" json:"category,omitempty"`
	Focus            []MessageDefinitionFocusComponent           `bson:"focus,omitempty" json:"focus,omitempty"`
	ResponseRequired *bool                                       `bson:"responseRequired,omitempty" json:"responseRequired,omitempty"`
	AllowedResponse  []MessageDefinitionAllowedResponseComponent `bson:"allowedResponse,omitempty" json:"allowedResponse,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *MessageDefinition) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "MessageDefinition"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to MessageDefinition), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *MessageDefinition) GetBSON() (interface{}, error) {
	x.ResourceType = "MessageDefinition"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "messageDefinition" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type messageDefinition MessageDefinition

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *MessageDefinition) UnmarshalJSON(data []byte) (err error) {
	x2 := messageDefinition{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = MessageDefinition(x2)
		return x.checkResourceType()
	}
	return
}

func (x *MessageDefinition) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "MessageDefinition"
	} else if x.ResourceType != "MessageDefinition" {
		return errors.New(fmt.Sprintf("Expected resourceType to be MessageDefinition, instead received %s", x.ResourceType))
	}
	return nil
}

type MessageDefinitionFocusComponent struct {
	BackboneElement `bson:",inline"`
	Code            string     `bson:"code,omitempty" json:"code,omitempty"`
	Profile         *Reference `bson:"profile,omitempty" json:"profile,omitempty"`
	Min             *uint32    `bson:"min,omitempty" json:"min,omitempty"`
	Max             string     `bson:"max,omitempty" json:"max,omitempty"`
}

type MessageDefinitionAllowedResponseComponent struct {
	BackboneElement `bson:",inline"`
	Message         *Reference `bson:"message,omitempty" json:"message,omitempty"`
	Situation       string     `bson:"situation,omitempty" json:"situation,omitempty"`
}
