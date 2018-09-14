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

type Condition struct {
	DomainResource     `bson:",inline"`
	Identifier         []Identifier                 `bson:"identifier,omitempty" json:"identifier,omitempty"`
	ClinicalStatus     string                       `bson:"clinicalStatus,omitempty" json:"clinicalStatus,omitempty"`
	VerificationStatus string                       `bson:"verificationStatus,omitempty" json:"verificationStatus,omitempty"`
	Category           []CodeableConcept            `bson:"category,omitempty" json:"category,omitempty"`
	Severity           *CodeableConcept             `bson:"severity,omitempty" json:"severity,omitempty"`
	Code               *CodeableConcept             `bson:"code,omitempty" json:"code,omitempty"`
	BodySite           []CodeableConcept            `bson:"bodySite,omitempty" json:"bodySite,omitempty"`
	Subject            *Reference                   `bson:"subject,omitempty" json:"subject,omitempty"`
	Context            *Reference                   `bson:"context,omitempty" json:"context,omitempty"`
	OnsetDateTime      *FHIRDateTime                `bson:"onsetDateTime,omitempty" json:"onsetDateTime,omitempty"`
	OnsetAge           *Quantity                    `bson:"onsetAge,omitempty" json:"onsetAge,omitempty"`
	OnsetPeriod        *Period                      `bson:"onsetPeriod,omitempty" json:"onsetPeriod,omitempty"`
	OnsetRange         *Range                       `bson:"onsetRange,omitempty" json:"onsetRange,omitempty"`
	OnsetString        string                       `bson:"onsetString,omitempty" json:"onsetString,omitempty"`
	AbatementDateTime  *FHIRDateTime                `bson:"abatementDateTime,omitempty" json:"abatementDateTime,omitempty"`
	AbatementAge       *Quantity                    `bson:"abatementAge,omitempty" json:"abatementAge,omitempty"`
	AbatementBoolean   *bool                        `bson:"abatementBoolean,omitempty" json:"abatementBoolean,omitempty"`
	AbatementPeriod    *Period                      `bson:"abatementPeriod,omitempty" json:"abatementPeriod,omitempty"`
	AbatementRange     *Range                       `bson:"abatementRange,omitempty" json:"abatementRange,omitempty"`
	AbatementString    string                       `bson:"abatementString,omitempty" json:"abatementString,omitempty"`
	AssertedDate       *FHIRDateTime                `bson:"assertedDate,omitempty" json:"assertedDate,omitempty"`
	Asserter           *Reference                   `bson:"asserter,omitempty" json:"asserter,omitempty"`
	Stage              *ConditionStageComponent     `bson:"stage,omitempty" json:"stage,omitempty"`
	Evidence           []ConditionEvidenceComponent `bson:"evidence,omitempty" json:"evidence,omitempty"`
	Note               []Annotation                 `bson:"note,omitempty" json:"note,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Condition) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Condition"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Condition), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Condition) GetBSON() (interface{}, error) {
	x.ResourceType = "Condition"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "condition" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type condition Condition

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Condition) UnmarshalJSON(data []byte) (err error) {
	x2 := condition{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Condition(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Condition) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Condition"
	} else if x.ResourceType != "Condition" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Condition, instead received %s", x.ResourceType))
	}
	return nil
}

type ConditionStageComponent struct {
	BackboneElement `bson:",inline"`
	Summary         *CodeableConcept `bson:"summary,omitempty" json:"summary,omitempty"`
	Assessment      []Reference      `bson:"assessment,omitempty" json:"assessment,omitempty"`
}

type ConditionEvidenceComponent struct {
	BackboneElement `bson:",inline"`
	Code            []CodeableConcept `bson:"code,omitempty" json:"code,omitempty"`
	Detail          []Reference       `bson:"detail,omitempty" json:"detail,omitempty"`
}
