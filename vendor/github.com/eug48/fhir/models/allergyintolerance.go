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

type AllergyIntolerance struct {
	DomainResource     `bson:",inline"`
	Identifier         []Identifier                          `bson:"identifier,omitempty" json:"identifier,omitempty"`
	ClinicalStatus     string                                `bson:"clinicalStatus,omitempty" json:"clinicalStatus,omitempty"`
	VerificationStatus string                                `bson:"verificationStatus,omitempty" json:"verificationStatus,omitempty"`
	Type               string                                `bson:"type,omitempty" json:"type,omitempty"`
	Category           []string                              `bson:"category,omitempty" json:"category,omitempty"`
	Criticality        string                                `bson:"criticality,omitempty" json:"criticality,omitempty"`
	Code               *CodeableConcept                      `bson:"code,omitempty" json:"code,omitempty"`
	Patient            *Reference                            `bson:"patient,omitempty" json:"patient,omitempty"`
	OnsetDateTime      *FHIRDateTime                         `bson:"onsetDateTime,omitempty" json:"onsetDateTime,omitempty"`
	OnsetAge           *Quantity                             `bson:"onsetAge,omitempty" json:"onsetAge,omitempty"`
	OnsetPeriod        *Period                               `bson:"onsetPeriod,omitempty" json:"onsetPeriod,omitempty"`
	OnsetRange         *Range                                `bson:"onsetRange,omitempty" json:"onsetRange,omitempty"`
	OnsetString        string                                `bson:"onsetString,omitempty" json:"onsetString,omitempty"`
	AssertedDate       *FHIRDateTime                         `bson:"assertedDate,omitempty" json:"assertedDate,omitempty"`
	Recorder           *Reference                            `bson:"recorder,omitempty" json:"recorder,omitempty"`
	Asserter           *Reference                            `bson:"asserter,omitempty" json:"asserter,omitempty"`
	LastOccurrence     *FHIRDateTime                         `bson:"lastOccurrence,omitempty" json:"lastOccurrence,omitempty"`
	Note               []Annotation                          `bson:"note,omitempty" json:"note,omitempty"`
	Reaction           []AllergyIntoleranceReactionComponent `bson:"reaction,omitempty" json:"reaction,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *AllergyIntolerance) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "AllergyIntolerance"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to AllergyIntolerance), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *AllergyIntolerance) GetBSON() (interface{}, error) {
	x.ResourceType = "AllergyIntolerance"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "allergyIntolerance" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type allergyIntolerance AllergyIntolerance

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *AllergyIntolerance) UnmarshalJSON(data []byte) (err error) {
	x2 := allergyIntolerance{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = AllergyIntolerance(x2)
		return x.checkResourceType()
	}
	return
}

func (x *AllergyIntolerance) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "AllergyIntolerance"
	} else if x.ResourceType != "AllergyIntolerance" {
		return errors.New(fmt.Sprintf("Expected resourceType to be AllergyIntolerance, instead received %s", x.ResourceType))
	}
	return nil
}

type AllergyIntoleranceReactionComponent struct {
	BackboneElement `bson:",inline"`
	Substance       *CodeableConcept  `bson:"substance,omitempty" json:"substance,omitempty"`
	Manifestation   []CodeableConcept `bson:"manifestation,omitempty" json:"manifestation,omitempty"`
	Description     string            `bson:"description,omitempty" json:"description,omitempty"`
	Onset           *FHIRDateTime     `bson:"onset,omitempty" json:"onset,omitempty"`
	Severity        string            `bson:"severity,omitempty" json:"severity,omitempty"`
	ExposureRoute   *CodeableConcept  `bson:"exposureRoute,omitempty" json:"exposureRoute,omitempty"`
	Note            []Annotation      `bson:"note,omitempty" json:"note,omitempty"`
}
