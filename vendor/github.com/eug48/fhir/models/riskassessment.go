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

type RiskAssessment struct {
	DomainResource        `bson:",inline"`
	Identifier            *Identifier                         `bson:"identifier,omitempty" json:"identifier,omitempty"`
	BasedOn               *Reference                          `bson:"basedOn,omitempty" json:"basedOn,omitempty"`
	Parent                *Reference                          `bson:"parent,omitempty" json:"parent,omitempty"`
	Status                string                              `bson:"status,omitempty" json:"status,omitempty"`
	Method                *CodeableConcept                    `bson:"method,omitempty" json:"method,omitempty"`
	Code                  *CodeableConcept                    `bson:"code,omitempty" json:"code,omitempty"`
	Subject               *Reference                          `bson:"subject,omitempty" json:"subject,omitempty"`
	Context               *Reference                          `bson:"context,omitempty" json:"context,omitempty"`
	OccurrenceDateTime    *FHIRDateTime                       `bson:"occurrenceDateTime,omitempty" json:"occurrenceDateTime,omitempty"`
	OccurrencePeriod      *Period                             `bson:"occurrencePeriod,omitempty" json:"occurrencePeriod,omitempty"`
	Condition             *Reference                          `bson:"condition,omitempty" json:"condition,omitempty"`
	Performer             *Reference                          `bson:"performer,omitempty" json:"performer,omitempty"`
	ReasonCodeableConcept *CodeableConcept                    `bson:"reasonCodeableConcept,omitempty" json:"reasonCodeableConcept,omitempty"`
	ReasonReference       *Reference                          `bson:"reasonReference,omitempty" json:"reasonReference,omitempty"`
	Basis                 []Reference                         `bson:"basis,omitempty" json:"basis,omitempty"`
	Prediction            []RiskAssessmentPredictionComponent `bson:"prediction,omitempty" json:"prediction,omitempty"`
	Mitigation            string                              `bson:"mitigation,omitempty" json:"mitigation,omitempty"`
	Comment               string                              `bson:"comment,omitempty" json:"comment,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *RiskAssessment) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "RiskAssessment"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to RiskAssessment), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *RiskAssessment) GetBSON() (interface{}, error) {
	x.ResourceType = "RiskAssessment"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "riskAssessment" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type riskAssessment RiskAssessment

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *RiskAssessment) UnmarshalJSON(data []byte) (err error) {
	x2 := riskAssessment{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = RiskAssessment(x2)
		return x.checkResourceType()
	}
	return
}

func (x *RiskAssessment) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "RiskAssessment"
	} else if x.ResourceType != "RiskAssessment" {
		return errors.New(fmt.Sprintf("Expected resourceType to be RiskAssessment, instead received %s", x.ResourceType))
	}
	return nil
}

type RiskAssessmentPredictionComponent struct {
	BackboneElement    `bson:",inline"`
	Outcome            *CodeableConcept `bson:"outcome,omitempty" json:"outcome,omitempty"`
	ProbabilityDecimal *float64         `bson:"probabilityDecimal,omitempty" json:"probabilityDecimal,omitempty"`
	ProbabilityRange   *Range           `bson:"probabilityRange,omitempty" json:"probabilityRange,omitempty"`
	QualitativeRisk    *CodeableConcept `bson:"qualitativeRisk,omitempty" json:"qualitativeRisk,omitempty"`
	RelativeRisk       *float64         `bson:"relativeRisk,omitempty" json:"relativeRisk,omitempty"`
	WhenPeriod         *Period          `bson:"whenPeriod,omitempty" json:"whenPeriod,omitempty"`
	WhenRange          *Range           `bson:"whenRange,omitempty" json:"whenRange,omitempty"`
	Rationale          string           `bson:"rationale,omitempty" json:"rationale,omitempty"`
}
