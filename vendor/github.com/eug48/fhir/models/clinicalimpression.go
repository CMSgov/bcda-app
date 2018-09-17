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

type ClinicalImpression struct {
	DomainResource           `bson:",inline"`
	Identifier               []Identifier                               `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status                   string                                     `bson:"status,omitempty" json:"status,omitempty"`
	Code                     *CodeableConcept                           `bson:"code,omitempty" json:"code,omitempty"`
	Description              string                                     `bson:"description,omitempty" json:"description,omitempty"`
	Subject                  *Reference                                 `bson:"subject,omitempty" json:"subject,omitempty"`
	Context                  *Reference                                 `bson:"context,omitempty" json:"context,omitempty"`
	EffectiveDateTime        *FHIRDateTime                              `bson:"effectiveDateTime,omitempty" json:"effectiveDateTime,omitempty"`
	EffectivePeriod          *Period                                    `bson:"effectivePeriod,omitempty" json:"effectivePeriod,omitempty"`
	Date                     *FHIRDateTime                              `bson:"date,omitempty" json:"date,omitempty"`
	Assessor                 *Reference                                 `bson:"assessor,omitempty" json:"assessor,omitempty"`
	Previous                 *Reference                                 `bson:"previous,omitempty" json:"previous,omitempty"`
	Problem                  []Reference                                `bson:"problem,omitempty" json:"problem,omitempty"`
	Investigation            []ClinicalImpressionInvestigationComponent `bson:"investigation,omitempty" json:"investigation,omitempty"`
	Protocol                 []string                                   `bson:"protocol,omitempty" json:"protocol,omitempty"`
	Summary                  string                                     `bson:"summary,omitempty" json:"summary,omitempty"`
	Finding                  []ClinicalImpressionFindingComponent       `bson:"finding,omitempty" json:"finding,omitempty"`
	PrognosisCodeableConcept []CodeableConcept                          `bson:"prognosisCodeableConcept,omitempty" json:"prognosisCodeableConcept,omitempty"`
	PrognosisReference       []Reference                                `bson:"prognosisReference,omitempty" json:"prognosisReference,omitempty"`
	Action                   []Reference                                `bson:"action,omitempty" json:"action,omitempty"`
	Note                     []Annotation                               `bson:"note,omitempty" json:"note,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *ClinicalImpression) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "ClinicalImpression"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to ClinicalImpression), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *ClinicalImpression) GetBSON() (interface{}, error) {
	x.ResourceType = "ClinicalImpression"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "clinicalImpression" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type clinicalImpression ClinicalImpression

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *ClinicalImpression) UnmarshalJSON(data []byte) (err error) {
	x2 := clinicalImpression{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = ClinicalImpression(x2)
		return x.checkResourceType()
	}
	return
}

func (x *ClinicalImpression) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "ClinicalImpression"
	} else if x.ResourceType != "ClinicalImpression" {
		return errors.New(fmt.Sprintf("Expected resourceType to be ClinicalImpression, instead received %s", x.ResourceType))
	}
	return nil
}

type ClinicalImpressionInvestigationComponent struct {
	BackboneElement `bson:",inline"`
	Code            *CodeableConcept `bson:"code,omitempty" json:"code,omitempty"`
	Item            []Reference      `bson:"item,omitempty" json:"item,omitempty"`
}

type ClinicalImpressionFindingComponent struct {
	BackboneElement     `bson:",inline"`
	ItemCodeableConcept *CodeableConcept `bson:"itemCodeableConcept,omitempty" json:"itemCodeableConcept,omitempty"`
	ItemReference       *Reference       `bson:"itemReference,omitempty" json:"itemReference,omitempty"`
	Basis               string           `bson:"basis,omitempty" json:"basis,omitempty"`
}
