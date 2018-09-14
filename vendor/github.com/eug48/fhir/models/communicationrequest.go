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

type CommunicationRequest struct {
	DomainResource     `bson:",inline"`
	Identifier         []Identifier                            `bson:"identifier,omitempty" json:"identifier,omitempty"`
	BasedOn            []Reference                             `bson:"basedOn,omitempty" json:"basedOn,omitempty"`
	Replaces           []Reference                             `bson:"replaces,omitempty" json:"replaces,omitempty"`
	GroupIdentifier    *Identifier                             `bson:"groupIdentifier,omitempty" json:"groupIdentifier,omitempty"`
	Status             string                                  `bson:"status,omitempty" json:"status,omitempty"`
	Category           []CodeableConcept                       `bson:"category,omitempty" json:"category,omitempty"`
	Priority           string                                  `bson:"priority,omitempty" json:"priority,omitempty"`
	Medium             []CodeableConcept                       `bson:"medium,omitempty" json:"medium,omitempty"`
	Subject            *Reference                              `bson:"subject,omitempty" json:"subject,omitempty"`
	Recipient          []Reference                             `bson:"recipient,omitempty" json:"recipient,omitempty"`
	Topic              []Reference                             `bson:"topic,omitempty" json:"topic,omitempty"`
	Context            *Reference                              `bson:"context,omitempty" json:"context,omitempty"`
	Payload            []CommunicationRequestPayloadComponent  `bson:"payload,omitempty" json:"payload,omitempty"`
	OccurrenceDateTime *FHIRDateTime                           `bson:"occurrenceDateTime,omitempty" json:"occurrenceDateTime,omitempty"`
	OccurrencePeriod   *Period                                 `bson:"occurrencePeriod,omitempty" json:"occurrencePeriod,omitempty"`
	AuthoredOn         *FHIRDateTime                           `bson:"authoredOn,omitempty" json:"authoredOn,omitempty"`
	Sender             *Reference                              `bson:"sender,omitempty" json:"sender,omitempty"`
	Requester          *CommunicationRequestRequesterComponent `bson:"requester,omitempty" json:"requester,omitempty"`
	ReasonCode         []CodeableConcept                       `bson:"reasonCode,omitempty" json:"reasonCode,omitempty"`
	ReasonReference    []Reference                             `bson:"reasonReference,omitempty" json:"reasonReference,omitempty"`
	Note               []Annotation                            `bson:"note,omitempty" json:"note,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *CommunicationRequest) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "CommunicationRequest"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to CommunicationRequest), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *CommunicationRequest) GetBSON() (interface{}, error) {
	x.ResourceType = "CommunicationRequest"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "communicationRequest" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type communicationRequest CommunicationRequest

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *CommunicationRequest) UnmarshalJSON(data []byte) (err error) {
	x2 := communicationRequest{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = CommunicationRequest(x2)
		return x.checkResourceType()
	}
	return
}

func (x *CommunicationRequest) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "CommunicationRequest"
	} else if x.ResourceType != "CommunicationRequest" {
		return errors.New(fmt.Sprintf("Expected resourceType to be CommunicationRequest, instead received %s", x.ResourceType))
	}
	return nil
}

type CommunicationRequestPayloadComponent struct {
	BackboneElement   `bson:",inline"`
	ContentString     string      `bson:"contentString,omitempty" json:"contentString,omitempty"`
	ContentAttachment *Attachment `bson:"contentAttachment,omitempty" json:"contentAttachment,omitempty"`
	ContentReference  *Reference  `bson:"contentReference,omitempty" json:"contentReference,omitempty"`
}

type CommunicationRequestRequesterComponent struct {
	BackboneElement `bson:",inline"`
	Agent           *Reference `bson:"agent,omitempty" json:"agent,omitempty"`
	OnBehalfOf      *Reference `bson:"onBehalfOf,omitempty" json:"onBehalfOf,omitempty"`
}
