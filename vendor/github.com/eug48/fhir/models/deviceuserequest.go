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

type DeviceUseRequest struct {
	DomainResource        `bson:",inline"`
	Identifier            []Identifier      `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Definition            []Reference       `bson:"definition,omitempty" json:"definition,omitempty"`
	BasedOn               []Reference       `bson:"basedOn,omitempty" json:"basedOn,omitempty"`
	Replaces              []Reference       `bson:"replaces,omitempty" json:"replaces,omitempty"`
	Requisition           *Identifier       `bson:"requisition,omitempty" json:"requisition,omitempty"`
	Status                string            `bson:"status,omitempty" json:"status,omitempty"`
	Stage                 *CodeableConcept  `bson:"stage,omitempty" json:"stage,omitempty"`
	DeviceReference       *Reference        `bson:"deviceReference,omitempty" json:"deviceReference,omitempty"`
	DeviceCodeableConcept *CodeableConcept  `bson:"deviceCodeableConcept,omitempty" json:"deviceCodeableConcept,omitempty"`
	Subject               *Reference        `bson:"subject,omitempty" json:"subject,omitempty"`
	Context               *Reference        `bson:"context,omitempty" json:"context,omitempty"`
	OccurrenceDateTime    *FHIRDateTime     `bson:"occurrenceDateTime,omitempty" json:"occurrenceDateTime,omitempty"`
	OccurrencePeriod      *Period           `bson:"occurrencePeriod,omitempty" json:"occurrencePeriod,omitempty"`
	OccurrenceTiming      *Timing           `bson:"occurrenceTiming,omitempty" json:"occurrenceTiming,omitempty"`
	Authored              *FHIRDateTime     `bson:"authored,omitempty" json:"authored,omitempty"`
	Requester             *Reference        `bson:"requester,omitempty" json:"requester,omitempty"`
	PerformerType         *CodeableConcept  `bson:"performerType,omitempty" json:"performerType,omitempty"`
	Performer             *Reference        `bson:"performer,omitempty" json:"performer,omitempty"`
	ReasonCode            []CodeableConcept `bson:"reasonCode,omitempty" json:"reasonCode,omitempty"`
	ReasonReference       []Reference       `bson:"reasonReference,omitempty" json:"reasonReference,omitempty"`
	SupportingInfo        []Reference       `bson:"supportingInfo,omitempty" json:"supportingInfo,omitempty"`
	Note                  []Annotation      `bson:"note,omitempty" json:"note,omitempty"`
	RelevantHistory       []Reference       `bson:"relevantHistory,omitempty" json:"relevantHistory,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *DeviceUseRequest) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "DeviceUseRequest"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to DeviceUseRequest), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *DeviceUseRequest) GetBSON() (interface{}, error) {
	x.ResourceType = "DeviceUseRequest"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "deviceUseRequest" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type deviceUseRequest DeviceUseRequest

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *DeviceUseRequest) UnmarshalJSON(data []byte) (err error) {
	x2 := deviceUseRequest{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = DeviceUseRequest(x2)
		return x.checkResourceType()
	}
	return
}

func (x *DeviceUseRequest) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "DeviceUseRequest"
	} else if x.ResourceType != "DeviceUseRequest" {
		return errors.New(fmt.Sprintf("Expected resourceType to be DeviceUseRequest, instead received %s", x.ResourceType))
	}
	return nil
}
