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

type MedicationAdministration struct {
	DomainResource            `bson:",inline"`
	Identifier                []Identifier                                 `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Definition                []Reference                                  `bson:"definition,omitempty" json:"definition,omitempty"`
	PartOf                    []Reference                                  `bson:"partOf,omitempty" json:"partOf,omitempty"`
	Status                    string                                       `bson:"status,omitempty" json:"status,omitempty"`
	Category                  *CodeableConcept                             `bson:"category,omitempty" json:"category,omitempty"`
	MedicationCodeableConcept *CodeableConcept                             `bson:"medicationCodeableConcept,omitempty" json:"medicationCodeableConcept,omitempty"`
	MedicationReference       *Reference                                   `bson:"medicationReference,omitempty" json:"medicationReference,omitempty"`
	Subject                   *Reference                                   `bson:"subject,omitempty" json:"subject,omitempty"`
	Context                   *Reference                                   `bson:"context,omitempty" json:"context,omitempty"`
	SupportingInformation     []Reference                                  `bson:"supportingInformation,omitempty" json:"supportingInformation,omitempty"`
	EffectiveDateTime         *FHIRDateTime                                `bson:"effectiveDateTime,omitempty" json:"effectiveDateTime,omitempty"`
	EffectivePeriod           *Period                                      `bson:"effectivePeriod,omitempty" json:"effectivePeriod,omitempty"`
	Performer                 []MedicationAdministrationPerformerComponent `bson:"performer,omitempty" json:"performer,omitempty"`
	NotGiven                  *bool                                        `bson:"notGiven,omitempty" json:"notGiven,omitempty"`
	ReasonNotGiven            []CodeableConcept                            `bson:"reasonNotGiven,omitempty" json:"reasonNotGiven,omitempty"`
	ReasonCode                []CodeableConcept                            `bson:"reasonCode,omitempty" json:"reasonCode,omitempty"`
	ReasonReference           []Reference                                  `bson:"reasonReference,omitempty" json:"reasonReference,omitempty"`
	Prescription              *Reference                                   `bson:"prescription,omitempty" json:"prescription,omitempty"`
	Device                    []Reference                                  `bson:"device,omitempty" json:"device,omitempty"`
	Note                      []Annotation                                 `bson:"note,omitempty" json:"note,omitempty"`
	Dosage                    *MedicationAdministrationDosageComponent     `bson:"dosage,omitempty" json:"dosage,omitempty"`
	EventHistory              []Reference                                  `bson:"eventHistory,omitempty" json:"eventHistory,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *MedicationAdministration) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "MedicationAdministration"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to MedicationAdministration), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *MedicationAdministration) GetBSON() (interface{}, error) {
	x.ResourceType = "MedicationAdministration"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "medicationAdministration" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type medicationAdministration MedicationAdministration

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *MedicationAdministration) UnmarshalJSON(data []byte) (err error) {
	x2 := medicationAdministration{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = MedicationAdministration(x2)
		return x.checkResourceType()
	}
	return
}

func (x *MedicationAdministration) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "MedicationAdministration"
	} else if x.ResourceType != "MedicationAdministration" {
		return errors.New(fmt.Sprintf("Expected resourceType to be MedicationAdministration, instead received %s", x.ResourceType))
	}
	return nil
}

type MedicationAdministrationPerformerComponent struct {
	BackboneElement `bson:",inline"`
	Actor           *Reference `bson:"actor,omitempty" json:"actor,omitempty"`
	OnBehalfOf      *Reference `bson:"onBehalfOf,omitempty" json:"onBehalfOf,omitempty"`
}

type MedicationAdministrationDosageComponent struct {
	BackboneElement    `bson:",inline"`
	Text               string           `bson:"text,omitempty" json:"text,omitempty"`
	Site               *CodeableConcept `bson:"site,omitempty" json:"site,omitempty"`
	Route              *CodeableConcept `bson:"route,omitempty" json:"route,omitempty"`
	Method             *CodeableConcept `bson:"method,omitempty" json:"method,omitempty"`
	Dose               *Quantity        `bson:"dose,omitempty" json:"dose,omitempty"`
	RateRatio          *Ratio           `bson:"rateRatio,omitempty" json:"rateRatio,omitempty"`
	RateSimpleQuantity *Quantity        `bson:"rateSimpleQuantity,omitempty" json:"rateSimpleQuantity,omitempty"`
}
