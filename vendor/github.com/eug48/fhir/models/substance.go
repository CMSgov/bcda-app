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

type Substance struct {
	DomainResource `bson:",inline"`
	Identifier     []Identifier                   `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status         string                         `bson:"status,omitempty" json:"status,omitempty"`
	Category       []CodeableConcept              `bson:"category,omitempty" json:"category,omitempty"`
	Code           *CodeableConcept               `bson:"code,omitempty" json:"code,omitempty"`
	Description    string                         `bson:"description,omitempty" json:"description,omitempty"`
	Instance       []SubstanceInstanceComponent   `bson:"instance,omitempty" json:"instance,omitempty"`
	Ingredient     []SubstanceIngredientComponent `bson:"ingredient,omitempty" json:"ingredient,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Substance) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Substance"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Substance), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Substance) GetBSON() (interface{}, error) {
	x.ResourceType = "Substance"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "substance" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type substance Substance

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Substance) UnmarshalJSON(data []byte) (err error) {
	x2 := substance{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Substance(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Substance) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Substance"
	} else if x.ResourceType != "Substance" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Substance, instead received %s", x.ResourceType))
	}
	return nil
}

type SubstanceInstanceComponent struct {
	BackboneElement `bson:",inline"`
	Identifier      *Identifier   `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Expiry          *FHIRDateTime `bson:"expiry,omitempty" json:"expiry,omitempty"`
	Quantity        *Quantity     `bson:"quantity,omitempty" json:"quantity,omitempty"`
}

type SubstanceIngredientComponent struct {
	BackboneElement          `bson:",inline"`
	Quantity                 *Ratio           `bson:"quantity,omitempty" json:"quantity,omitempty"`
	SubstanceCodeableConcept *CodeableConcept `bson:"substanceCodeableConcept,omitempty" json:"substanceCodeableConcept,omitempty"`
	SubstanceReference       *Reference       `bson:"substanceReference,omitempty" json:"substanceReference,omitempty"`
}
