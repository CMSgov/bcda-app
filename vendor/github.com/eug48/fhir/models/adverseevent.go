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

type AdverseEvent struct {
	DomainResource        `bson:",inline"`
	Identifier            *Identifier                          `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Category              string                               `bson:"category,omitempty" json:"category,omitempty"`
	Type                  *CodeableConcept                     `bson:"type,omitempty" json:"type,omitempty"`
	Subject               *Reference                           `bson:"subject,omitempty" json:"subject,omitempty"`
	Date                  *FHIRDateTime                        `bson:"date,omitempty" json:"date,omitempty"`
	Reaction              []Reference                          `bson:"reaction,omitempty" json:"reaction,omitempty"`
	Location              *Reference                           `bson:"location,omitempty" json:"location,omitempty"`
	Seriousness           *CodeableConcept                     `bson:"seriousness,omitempty" json:"seriousness,omitempty"`
	Outcome               *CodeableConcept                     `bson:"outcome,omitempty" json:"outcome,omitempty"`
	Recorder              *Reference                           `bson:"recorder,omitempty" json:"recorder,omitempty"`
	EventParticipant      *Reference                           `bson:"eventParticipant,omitempty" json:"eventParticipant,omitempty"`
	Description           string                               `bson:"description,omitempty" json:"description,omitempty"`
	SuspectEntity         []AdverseEventSuspectEntityComponent `bson:"suspectEntity,omitempty" json:"suspectEntity,omitempty"`
	SubjectMedicalHistory []Reference                          `bson:"subjectMedicalHistory,omitempty" json:"subjectMedicalHistory,omitempty"`
	ReferenceDocument     []Reference                          `bson:"referenceDocument,omitempty" json:"referenceDocument,omitempty"`
	Study                 []Reference                          `bson:"study,omitempty" json:"study,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *AdverseEvent) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "AdverseEvent"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to AdverseEvent), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *AdverseEvent) GetBSON() (interface{}, error) {
	x.ResourceType = "AdverseEvent"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "adverseEvent" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type adverseEvent AdverseEvent

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *AdverseEvent) UnmarshalJSON(data []byte) (err error) {
	x2 := adverseEvent{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = AdverseEvent(x2)
		return x.checkResourceType()
	}
	return
}

func (x *AdverseEvent) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "AdverseEvent"
	} else if x.ResourceType != "AdverseEvent" {
		return errors.New(fmt.Sprintf("Expected resourceType to be AdverseEvent, instead received %s", x.ResourceType))
	}
	return nil
}

type AdverseEventSuspectEntityComponent struct {
	BackboneElement             `bson:",inline"`
	Instance                    *Reference       `bson:"instance,omitempty" json:"instance,omitempty"`
	Causality                   string           `bson:"causality,omitempty" json:"causality,omitempty"`
	CausalityAssessment         *CodeableConcept `bson:"causalityAssessment,omitempty" json:"causalityAssessment,omitempty"`
	CausalityProductRelatedness string           `bson:"causalityProductRelatedness,omitempty" json:"causalityProductRelatedness,omitempty"`
	CausalityMethod             *CodeableConcept `bson:"causalityMethod,omitempty" json:"causalityMethod,omitempty"`
	CausalityAuthor             *Reference       `bson:"causalityAuthor,omitempty" json:"causalityAuthor,omitempty"`
	CausalityResult             *CodeableConcept `bson:"causalityResult,omitempty" json:"causalityResult,omitempty"`
}
