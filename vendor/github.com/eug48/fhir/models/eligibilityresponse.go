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

type EligibilityResponse struct {
	DomainResource      `bson:",inline"`
	Identifier          []Identifier                            `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status              string                                  `bson:"status,omitempty" json:"status,omitempty"`
	Created             *FHIRDateTime                           `bson:"created,omitempty" json:"created,omitempty"`
	RequestProvider     *Reference                              `bson:"requestProvider,omitempty" json:"requestProvider,omitempty"`
	RequestOrganization *Reference                              `bson:"requestOrganization,omitempty" json:"requestOrganization,omitempty"`
	Request             *Reference                              `bson:"request,omitempty" json:"request,omitempty"`
	Outcome             *CodeableConcept                        `bson:"outcome,omitempty" json:"outcome,omitempty"`
	Disposition         string                                  `bson:"disposition,omitempty" json:"disposition,omitempty"`
	Insurer             *Reference                              `bson:"insurer,omitempty" json:"insurer,omitempty"`
	Inforce             *bool                                   `bson:"inforce,omitempty" json:"inforce,omitempty"`
	Insurance           []EligibilityResponseInsuranceComponent `bson:"insurance,omitempty" json:"insurance,omitempty"`
	Form                *CodeableConcept                        `bson:"form,omitempty" json:"form,omitempty"`
	Error               []EligibilityResponseErrorsComponent    `bson:"error,omitempty" json:"error,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *EligibilityResponse) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "EligibilityResponse"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to EligibilityResponse), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *EligibilityResponse) GetBSON() (interface{}, error) {
	x.ResourceType = "EligibilityResponse"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "eligibilityResponse" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type eligibilityResponse EligibilityResponse

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *EligibilityResponse) UnmarshalJSON(data []byte) (err error) {
	x2 := eligibilityResponse{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = EligibilityResponse(x2)
		return x.checkResourceType()
	}
	return
}

func (x *EligibilityResponse) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "EligibilityResponse"
	} else if x.ResourceType != "EligibilityResponse" {
		return errors.New(fmt.Sprintf("Expected resourceType to be EligibilityResponse, instead received %s", x.ResourceType))
	}
	return nil
}

type EligibilityResponseInsuranceComponent struct {
	BackboneElement `bson:",inline"`
	Coverage        *Reference                             `bson:"coverage,omitempty" json:"coverage,omitempty"`
	Contract        *Reference                             `bson:"contract,omitempty" json:"contract,omitempty"`
	BenefitBalance  []EligibilityResponseBenefitsComponent `bson:"benefitBalance,omitempty" json:"benefitBalance,omitempty"`
}

type EligibilityResponseBenefitsComponent struct {
	BackboneElement `bson:",inline"`
	Category        *CodeableConcept                      `bson:"category,omitempty" json:"category,omitempty"`
	SubCategory     *CodeableConcept                      `bson:"subCategory,omitempty" json:"subCategory,omitempty"`
	Excluded        *bool                                 `bson:"excluded,omitempty" json:"excluded,omitempty"`
	Name            string                                `bson:"name,omitempty" json:"name,omitempty"`
	Description     string                                `bson:"description,omitempty" json:"description,omitempty"`
	Network         *CodeableConcept                      `bson:"network,omitempty" json:"network,omitempty"`
	Unit            *CodeableConcept                      `bson:"unit,omitempty" json:"unit,omitempty"`
	Term            *CodeableConcept                      `bson:"term,omitempty" json:"term,omitempty"`
	Financial       []EligibilityResponseBenefitComponent `bson:"financial,omitempty" json:"financial,omitempty"`
}

type EligibilityResponseBenefitComponent struct {
	BackboneElement    `bson:",inline"`
	Type               *CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	AllowedUnsignedInt *uint32          `bson:"allowedUnsignedInt,omitempty" json:"allowedUnsignedInt,omitempty"`
	AllowedString      string           `bson:"allowedString,omitempty" json:"allowedString,omitempty"`
	AllowedMoney       *Quantity        `bson:"allowedMoney,omitempty" json:"allowedMoney,omitempty"`
	UsedUnsignedInt    *uint32          `bson:"usedUnsignedInt,omitempty" json:"usedUnsignedInt,omitempty"`
	UsedMoney          *Quantity        `bson:"usedMoney,omitempty" json:"usedMoney,omitempty"`
}

type EligibilityResponseErrorsComponent struct {
	BackboneElement `bson:",inline"`
	Code            *CodeableConcept `bson:"code,omitempty" json:"code,omitempty"`
}
