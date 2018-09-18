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

type EligibilityRequest struct {
	DomainResource      `bson:",inline"`
	Identifier          []Identifier     `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status              string           `bson:"status,omitempty" json:"status,omitempty"`
	Priority            *CodeableConcept `bson:"priority,omitempty" json:"priority,omitempty"`
	Patient             *Reference       `bson:"patient,omitempty" json:"patient,omitempty"`
	ServicedDate        *FHIRDateTime    `bson:"servicedDate,omitempty" json:"servicedDate,omitempty"`
	ServicedPeriod      *Period          `bson:"servicedPeriod,omitempty" json:"servicedPeriod,omitempty"`
	Created             *FHIRDateTime    `bson:"created,omitempty" json:"created,omitempty"`
	Enterer             *Reference       `bson:"enterer,omitempty" json:"enterer,omitempty"`
	Provider            *Reference       `bson:"provider,omitempty" json:"provider,omitempty"`
	Organization        *Reference       `bson:"organization,omitempty" json:"organization,omitempty"`
	Insurer             *Reference       `bson:"insurer,omitempty" json:"insurer,omitempty"`
	Facility            *Reference       `bson:"facility,omitempty" json:"facility,omitempty"`
	Coverage            *Reference       `bson:"coverage,omitempty" json:"coverage,omitempty"`
	BusinessArrangement string           `bson:"businessArrangement,omitempty" json:"businessArrangement,omitempty"`
	BenefitCategory     *CodeableConcept `bson:"benefitCategory,omitempty" json:"benefitCategory,omitempty"`
	BenefitSubCategory  *CodeableConcept `bson:"benefitSubCategory,omitempty" json:"benefitSubCategory,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *EligibilityRequest) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "EligibilityRequest"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to EligibilityRequest), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *EligibilityRequest) GetBSON() (interface{}, error) {
	x.ResourceType = "EligibilityRequest"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "eligibilityRequest" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type eligibilityRequest EligibilityRequest

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *EligibilityRequest) UnmarshalJSON(data []byte) (err error) {
	x2 := eligibilityRequest{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = EligibilityRequest(x2)
		return x.checkResourceType()
	}
	return
}

func (x *EligibilityRequest) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "EligibilityRequest"
	} else if x.ResourceType != "EligibilityRequest" {
		return errors.New(fmt.Sprintf("Expected resourceType to be EligibilityRequest, instead received %s", x.ResourceType))
	}
	return nil
}
