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

type NutritionRequest struct {
	DomainResource         `bson:",inline"`
	Identifier             []Identifier                             `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status                 string                                   `bson:"status,omitempty" json:"status,omitempty"`
	Patient                *Reference                               `bson:"patient,omitempty" json:"patient,omitempty"`
	Encounter              *Reference                               `bson:"encounter,omitempty" json:"encounter,omitempty"`
	DateTime               *FHIRDateTime                            `bson:"dateTime,omitempty" json:"dateTime,omitempty"`
	Orderer                *Reference                               `bson:"orderer,omitempty" json:"orderer,omitempty"`
	AllergyIntolerance     []Reference                              `bson:"allergyIntolerance,omitempty" json:"allergyIntolerance,omitempty"`
	FoodPreferenceModifier []CodeableConcept                        `bson:"foodPreferenceModifier,omitempty" json:"foodPreferenceModifier,omitempty"`
	ExcludeFoodModifier    []CodeableConcept                        `bson:"excludeFoodModifier,omitempty" json:"excludeFoodModifier,omitempty"`
	OralDiet               *NutritionRequestOralDietComponent       `bson:"oralDiet,omitempty" json:"oralDiet,omitempty"`
	Supplement             []NutritionRequestSupplementComponent    `bson:"supplement,omitempty" json:"supplement,omitempty"`
	EnteralFormula         *NutritionRequestEnteralFormulaComponent `bson:"enteralFormula,omitempty" json:"enteralFormula,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *NutritionRequest) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "NutritionRequest"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to NutritionRequest), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *NutritionRequest) GetBSON() (interface{}, error) {
	x.ResourceType = "NutritionRequest"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "nutritionRequest" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type nutritionRequest NutritionRequest

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *NutritionRequest) UnmarshalJSON(data []byte) (err error) {
	x2 := nutritionRequest{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = NutritionRequest(x2)
		return x.checkResourceType()
	}
	return
}

func (x *NutritionRequest) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "NutritionRequest"
	} else if x.ResourceType != "NutritionRequest" {
		return errors.New(fmt.Sprintf("Expected resourceType to be NutritionRequest, instead received %s", x.ResourceType))
	}
	return nil
}

type NutritionRequestOralDietComponent struct {
	BackboneElement      `bson:",inline"`
	Type                 []CodeableConcept                           `bson:"type,omitempty" json:"type,omitempty"`
	Schedule             []Timing                                    `bson:"schedule,omitempty" json:"schedule,omitempty"`
	Nutrient             []NutritionRequestOralDietNutrientComponent `bson:"nutrient,omitempty" json:"nutrient,omitempty"`
	Texture              []NutritionRequestOralDietTextureComponent  `bson:"texture,omitempty" json:"texture,omitempty"`
	FluidConsistencyType []CodeableConcept                           `bson:"fluidConsistencyType,omitempty" json:"fluidConsistencyType,omitempty"`
	Instruction          string                                      `bson:"instruction,omitempty" json:"instruction,omitempty"`
}

type NutritionRequestOralDietNutrientComponent struct {
	BackboneElement `bson:",inline"`
	Modifier        *CodeableConcept `bson:"modifier,omitempty" json:"modifier,omitempty"`
	Amount          *Quantity        `bson:"amount,omitempty" json:"amount,omitempty"`
}

type NutritionRequestOralDietTextureComponent struct {
	BackboneElement `bson:",inline"`
	Modifier        *CodeableConcept `bson:"modifier,omitempty" json:"modifier,omitempty"`
	FoodType        *CodeableConcept `bson:"foodType,omitempty" json:"foodType,omitempty"`
}

type NutritionRequestSupplementComponent struct {
	BackboneElement `bson:",inline"`
	Type            *CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	ProductName     string           `bson:"productName,omitempty" json:"productName,omitempty"`
	Schedule        []Timing         `bson:"schedule,omitempty" json:"schedule,omitempty"`
	Quantity        *Quantity        `bson:"quantity,omitempty" json:"quantity,omitempty"`
	Instruction     string           `bson:"instruction,omitempty" json:"instruction,omitempty"`
}

type NutritionRequestEnteralFormulaComponent struct {
	BackboneElement           `bson:",inline"`
	BaseFormulaType           *CodeableConcept                                        `bson:"baseFormulaType,omitempty" json:"baseFormulaType,omitempty"`
	BaseFormulaProductName    string                                                  `bson:"baseFormulaProductName,omitempty" json:"baseFormulaProductName,omitempty"`
	AdditiveType              *CodeableConcept                                        `bson:"additiveType,omitempty" json:"additiveType,omitempty"`
	AdditiveProductName       string                                                  `bson:"additiveProductName,omitempty" json:"additiveProductName,omitempty"`
	CaloricDensity            *Quantity                                               `bson:"caloricDensity,omitempty" json:"caloricDensity,omitempty"`
	RouteofAdministration     *CodeableConcept                                        `bson:"routeofAdministration,omitempty" json:"routeofAdministration,omitempty"`
	Administration            []NutritionRequestEnteralFormulaAdministrationComponent `bson:"administration,omitempty" json:"administration,omitempty"`
	MaxVolumeToDeliver        *Quantity                                               `bson:"maxVolumeToDeliver,omitempty" json:"maxVolumeToDeliver,omitempty"`
	AdministrationInstruction string                                                  `bson:"administrationInstruction,omitempty" json:"administrationInstruction,omitempty"`
}

type NutritionRequestEnteralFormulaAdministrationComponent struct {
	BackboneElement    `bson:",inline"`
	Schedule           *Timing   `bson:"schedule,omitempty" json:"schedule,omitempty"`
	Quantity           *Quantity `bson:"quantity,omitempty" json:"quantity,omitempty"`
	RateSimpleQuantity *Quantity `bson:"rateSimpleQuantity,omitempty" json:"rateSimpleQuantity,omitempty"`
	RateRatio          *Ratio    `bson:"rateRatio,omitempty" json:"rateRatio,omitempty"`
}
