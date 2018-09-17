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

type Medication struct {
	DomainResource   `bson:",inline"`
	Code             *CodeableConcept                `bson:"code,omitempty" json:"code,omitempty"`
	Status           string                          `bson:"status,omitempty" json:"status,omitempty"`
	IsBrand          *bool                           `bson:"isBrand,omitempty" json:"isBrand,omitempty"`
	IsOverTheCounter *bool                           `bson:"isOverTheCounter,omitempty" json:"isOverTheCounter,omitempty"`
	Manufacturer     *Reference                      `bson:"manufacturer,omitempty" json:"manufacturer,omitempty"`
	Form             *CodeableConcept                `bson:"form,omitempty" json:"form,omitempty"`
	Ingredient       []MedicationIngredientComponent `bson:"ingredient,omitempty" json:"ingredient,omitempty"`
	Package          *MedicationPackageComponent     `bson:"package,omitempty" json:"package,omitempty"`
	Image            []Attachment                    `bson:"image,omitempty" json:"image,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Medication) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Medication"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Medication), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Medication) GetBSON() (interface{}, error) {
	x.ResourceType = "Medication"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "medication" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type medication Medication

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Medication) UnmarshalJSON(data []byte) (err error) {
	x2 := medication{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Medication(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Medication) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Medication"
	} else if x.ResourceType != "Medication" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Medication, instead received %s", x.ResourceType))
	}
	return nil
}

type MedicationIngredientComponent struct {
	BackboneElement     `bson:",inline"`
	ItemCodeableConcept *CodeableConcept `bson:"itemCodeableConcept,omitempty" json:"itemCodeableConcept,omitempty"`
	ItemReference       *Reference       `bson:"itemReference,omitempty" json:"itemReference,omitempty"`
	IsActive            *bool            `bson:"isActive,omitempty" json:"isActive,omitempty"`
	Amount              *Ratio           `bson:"amount,omitempty" json:"amount,omitempty"`
}

type MedicationPackageComponent struct {
	BackboneElement `bson:",inline"`
	Container       *CodeableConcept                    `bson:"container,omitempty" json:"container,omitempty"`
	Content         []MedicationPackageContentComponent `bson:"content,omitempty" json:"content,omitempty"`
	Batch           []MedicationPackageBatchComponent   `bson:"batch,omitempty" json:"batch,omitempty"`
}

type MedicationPackageContentComponent struct {
	BackboneElement     `bson:",inline"`
	ItemCodeableConcept *CodeableConcept `bson:"itemCodeableConcept,omitempty" json:"itemCodeableConcept,omitempty"`
	ItemReference       *Reference       `bson:"itemReference,omitempty" json:"itemReference,omitempty"`
	Amount              *Quantity        `bson:"amount,omitempty" json:"amount,omitempty"`
}

type MedicationPackageBatchComponent struct {
	BackboneElement `bson:",inline"`
	LotNumber       string        `bson:"lotNumber,omitempty" json:"lotNumber,omitempty"`
	ExpirationDate  *FHIRDateTime `bson:"expirationDate,omitempty" json:"expirationDate,omitempty"`
}
