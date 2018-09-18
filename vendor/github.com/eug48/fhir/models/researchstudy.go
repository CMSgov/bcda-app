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

type ResearchStudy struct {
	DomainResource        `bson:",inline"`
	Identifier            []Identifier                `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Title                 string                      `bson:"title,omitempty" json:"title,omitempty"`
	Protocol              []Reference                 `bson:"protocol,omitempty" json:"protocol,omitempty"`
	PartOf                []Reference                 `bson:"partOf,omitempty" json:"partOf,omitempty"`
	Status                string                      `bson:"status,omitempty" json:"status,omitempty"`
	Category              []CodeableConcept           `bson:"category,omitempty" json:"category,omitempty"`
	Focus                 []CodeableConcept           `bson:"focus,omitempty" json:"focus,omitempty"`
	Contact               []ContactDetail             `bson:"contact,omitempty" json:"contact,omitempty"`
	RelatedArtifact       []RelatedArtifact           `bson:"relatedArtifact,omitempty" json:"relatedArtifact,omitempty"`
	Keyword               []CodeableConcept           `bson:"keyword,omitempty" json:"keyword,omitempty"`
	Jurisdiction          []CodeableConcept           `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	Description           string                      `bson:"description,omitempty" json:"description,omitempty"`
	Enrollment            []Reference                 `bson:"enrollment,omitempty" json:"enrollment,omitempty"`
	Period                *Period                     `bson:"period,omitempty" json:"period,omitempty"`
	Sponsor               *Reference                  `bson:"sponsor,omitempty" json:"sponsor,omitempty"`
	PrincipalInvestigator *Reference                  `bson:"principalInvestigator,omitempty" json:"principalInvestigator,omitempty"`
	Site                  []Reference                 `bson:"site,omitempty" json:"site,omitempty"`
	ReasonStopped         *CodeableConcept            `bson:"reasonStopped,omitempty" json:"reasonStopped,omitempty"`
	Note                  []Annotation                `bson:"note,omitempty" json:"note,omitempty"`
	Arm                   []ResearchStudyArmComponent `bson:"arm,omitempty" json:"arm,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *ResearchStudy) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "ResearchStudy"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to ResearchStudy), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *ResearchStudy) GetBSON() (interface{}, error) {
	x.ResourceType = "ResearchStudy"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "researchStudy" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type researchStudy ResearchStudy

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *ResearchStudy) UnmarshalJSON(data []byte) (err error) {
	x2 := researchStudy{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = ResearchStudy(x2)
		return x.checkResourceType()
	}
	return
}

func (x *ResearchStudy) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "ResearchStudy"
	} else if x.ResourceType != "ResearchStudy" {
		return errors.New(fmt.Sprintf("Expected resourceType to be ResearchStudy, instead received %s", x.ResourceType))
	}
	return nil
}

type ResearchStudyArmComponent struct {
	BackboneElement `bson:",inline"`
	Name            string           `bson:"name,omitempty" json:"name,omitempty"`
	Code            *CodeableConcept `bson:"code,omitempty" json:"code,omitempty"`
	Description     string           `bson:"description,omitempty" json:"description,omitempty"`
}
