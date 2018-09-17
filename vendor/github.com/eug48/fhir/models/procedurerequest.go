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

type ProcedureRequest struct {
	DomainResource          `bson:",inline"`
	Identifier              []Identifier                        `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Definition              []Reference                         `bson:"definition,omitempty" json:"definition,omitempty"`
	BasedOn                 []Reference                         `bson:"basedOn,omitempty" json:"basedOn,omitempty"`
	Replaces                []Reference                         `bson:"replaces,omitempty" json:"replaces,omitempty"`
	Requisition             *Identifier                         `bson:"requisition,omitempty" json:"requisition,omitempty"`
	Status                  string                              `bson:"status,omitempty" json:"status,omitempty"`
	Intent                  string                              `bson:"intent,omitempty" json:"intent,omitempty"`
	Priority                string                              `bson:"priority,omitempty" json:"priority,omitempty"`
	DoNotPerform            *bool                               `bson:"doNotPerform,omitempty" json:"doNotPerform,omitempty"`
	Category                []CodeableConcept                   `bson:"category,omitempty" json:"category,omitempty"`
	Code                    *CodeableConcept                    `bson:"code,omitempty" json:"code,omitempty"`
	Subject                 *Reference                          `bson:"subject,omitempty" json:"subject,omitempty"`
	Context                 *Reference                          `bson:"context,omitempty" json:"context,omitempty"`
	OccurrenceDateTime      *FHIRDateTime                       `bson:"occurrenceDateTime,omitempty" json:"occurrenceDateTime,omitempty"`
	OccurrencePeriod        *Period                             `bson:"occurrencePeriod,omitempty" json:"occurrencePeriod,omitempty"`
	OccurrenceTiming        *Timing                             `bson:"occurrenceTiming,omitempty" json:"occurrenceTiming,omitempty"`
	AsNeededBoolean         *bool                               `bson:"asNeededBoolean,omitempty" json:"asNeededBoolean,omitempty"`
	AsNeededCodeableConcept *CodeableConcept                    `bson:"asNeededCodeableConcept,omitempty" json:"asNeededCodeableConcept,omitempty"`
	AuthoredOn              *FHIRDateTime                       `bson:"authoredOn,omitempty" json:"authoredOn,omitempty"`
	Requester               *ProcedureRequestRequesterComponent `bson:"requester,omitempty" json:"requester,omitempty"`
	PerformerType           *CodeableConcept                    `bson:"performerType,omitempty" json:"performerType,omitempty"`
	Performer               *Reference                          `bson:"performer,omitempty" json:"performer,omitempty"`
	ReasonCode              []CodeableConcept                   `bson:"reasonCode,omitempty" json:"reasonCode,omitempty"`
	ReasonReference         []Reference                         `bson:"reasonReference,omitempty" json:"reasonReference,omitempty"`
	SupportingInfo          []Reference                         `bson:"supportingInfo,omitempty" json:"supportingInfo,omitempty"`
	Specimen                []Reference                         `bson:"specimen,omitempty" json:"specimen,omitempty"`
	BodySite                []CodeableConcept                   `bson:"bodySite,omitempty" json:"bodySite,omitempty"`
	Note                    []Annotation                        `bson:"note,omitempty" json:"note,omitempty"`
	RelevantHistory         []Reference                         `bson:"relevantHistory,omitempty" json:"relevantHistory,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *ProcedureRequest) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "ProcedureRequest"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to ProcedureRequest), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *ProcedureRequest) GetBSON() (interface{}, error) {
	x.ResourceType = "ProcedureRequest"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "procedureRequest" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type procedureRequest ProcedureRequest

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *ProcedureRequest) UnmarshalJSON(data []byte) (err error) {
	x2 := procedureRequest{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = ProcedureRequest(x2)
		return x.checkResourceType()
	}
	return
}

func (x *ProcedureRequest) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "ProcedureRequest"
	} else if x.ResourceType != "ProcedureRequest" {
		return errors.New(fmt.Sprintf("Expected resourceType to be ProcedureRequest, instead received %s", x.ResourceType))
	}
	return nil
}

type ProcedureRequestRequesterComponent struct {
	BackboneElement `bson:",inline"`
	Agent           *Reference `bson:"agent,omitempty" json:"agent,omitempty"`
	OnBehalfOf      *Reference `bson:"onBehalfOf,omitempty" json:"onBehalfOf,omitempty"`
}
