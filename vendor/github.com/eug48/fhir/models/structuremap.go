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

type StructureMap struct {
	DomainResource `bson:",inline"`
	Url            string                           `bson:"url,omitempty" json:"url,omitempty"`
	Identifier     []Identifier                     `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Version        string                           `bson:"version,omitempty" json:"version,omitempty"`
	Name           string                           `bson:"name,omitempty" json:"name,omitempty"`
	Title          string                           `bson:"title,omitempty" json:"title,omitempty"`
	Status         string                           `bson:"status,omitempty" json:"status,omitempty"`
	Experimental   *bool                            `bson:"experimental,omitempty" json:"experimental,omitempty"`
	Date           *FHIRDateTime                    `bson:"date,omitempty" json:"date,omitempty"`
	Publisher      string                           `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Contact        []ContactDetail                  `bson:"contact,omitempty" json:"contact,omitempty"`
	Description    string                           `bson:"description,omitempty" json:"description,omitempty"`
	UseContext     []UsageContext                   `bson:"useContext,omitempty" json:"useContext,omitempty"`
	Jurisdiction   []CodeableConcept                `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	Purpose        string                           `bson:"purpose,omitempty" json:"purpose,omitempty"`
	Copyright      string                           `bson:"copyright,omitempty" json:"copyright,omitempty"`
	Structure      []StructureMapStructureComponent `bson:"structure,omitempty" json:"structure,omitempty"`
	Import         []string                         `bson:"import,omitempty" json:"import,omitempty"`
	Group          []StructureMapGroupComponent     `bson:"group,omitempty" json:"group,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *StructureMap) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "StructureMap"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to StructureMap), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *StructureMap) GetBSON() (interface{}, error) {
	x.ResourceType = "StructureMap"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "structureMap" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type structureMap StructureMap

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *StructureMap) UnmarshalJSON(data []byte) (err error) {
	x2 := structureMap{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = StructureMap(x2)
		return x.checkResourceType()
	}
	return
}

func (x *StructureMap) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "StructureMap"
	} else if x.ResourceType != "StructureMap" {
		return errors.New(fmt.Sprintf("Expected resourceType to be StructureMap, instead received %s", x.ResourceType))
	}
	return nil
}

type StructureMapStructureComponent struct {
	BackboneElement `bson:",inline"`
	Url             string `bson:"url,omitempty" json:"url,omitempty"`
	Mode            string `bson:"mode,omitempty" json:"mode,omitempty"`
	Alias           string `bson:"alias,omitempty" json:"alias,omitempty"`
	Documentation   string `bson:"documentation,omitempty" json:"documentation,omitempty"`
}

type StructureMapGroupComponent struct {
	BackboneElement `bson:",inline"`
	Name            string                            `bson:"name,omitempty" json:"name,omitempty"`
	Extends         string                            `bson:"extends,omitempty" json:"extends,omitempty"`
	TypeMode        string                            `bson:"typeMode,omitempty" json:"typeMode,omitempty"`
	Documentation   string                            `bson:"documentation,omitempty" json:"documentation,omitempty"`
	Input           []StructureMapGroupInputComponent `bson:"input,omitempty" json:"input,omitempty"`
	Rule            []StructureMapGroupRuleComponent  `bson:"rule,omitempty" json:"rule,omitempty"`
}

type StructureMapGroupInputComponent struct {
	BackboneElement `bson:",inline"`
	Name            string `bson:"name,omitempty" json:"name,omitempty"`
	Type            string `bson:"type,omitempty" json:"type,omitempty"`
	Mode            string `bson:"mode,omitempty" json:"mode,omitempty"`
	Documentation   string `bson:"documentation,omitempty" json:"documentation,omitempty"`
}

type StructureMapGroupRuleComponent struct {
	BackboneElement `bson:",inline"`
	Name            string                                    `bson:"name,omitempty" json:"name,omitempty"`
	Source          []StructureMapGroupRuleSourceComponent    `bson:"source,omitempty" json:"source,omitempty"`
	Target          []StructureMapGroupRuleTargetComponent    `bson:"target,omitempty" json:"target,omitempty"`
	Rule            []StructureMapGroupRuleComponent          `bson:"rule,omitempty" json:"rule,omitempty"`
	Dependent       []StructureMapGroupRuleDependentComponent `bson:"dependent,omitempty" json:"dependent,omitempty"`
	Documentation   string                                    `bson:"documentation,omitempty" json:"documentation,omitempty"`
}

type StructureMapGroupRuleSourceComponent struct {
	BackboneElement             `bson:",inline"`
	Context                     string           `bson:"context,omitempty" json:"context,omitempty"`
	Min                         *int32           `bson:"min,omitempty" json:"min,omitempty"`
	Max                         string           `bson:"max,omitempty" json:"max,omitempty"`
	Type                        string           `bson:"type,omitempty" json:"type,omitempty"`
	DefaultValueAddress         *Address         `bson:"defaultValueAddress,omitempty" json:"defaultValueAddress,omitempty"`
	DefaultValueAnnotation      *Annotation      `bson:"defaultValueAnnotation,omitempty" json:"defaultValueAnnotation,omitempty"`
	DefaultValueAttachment      *Attachment      `bson:"defaultValueAttachment,omitempty" json:"defaultValueAttachment,omitempty"`
	DefaultValueBase64Binary    string           `bson:"defaultValueBase64Binary,omitempty" json:"defaultValueBase64Binary,omitempty"`
	DefaultValueBoolean         *bool            `bson:"defaultValueBoolean,omitempty" json:"defaultValueBoolean,omitempty"`
	DefaultValueCode            string           `bson:"defaultValueCode,omitempty" json:"defaultValueCode,omitempty"`
	DefaultValueCodeableConcept *CodeableConcept `bson:"defaultValueCodeableConcept,omitempty" json:"defaultValueCodeableConcept,omitempty"`
	DefaultValueCoding          *Coding          `bson:"defaultValueCoding,omitempty" json:"defaultValueCoding,omitempty"`
	DefaultValueContactPoint    *ContactPoint    `bson:"defaultValueContactPoint,omitempty" json:"defaultValueContactPoint,omitempty"`
	DefaultValueDate            *FHIRDateTime    `bson:"defaultValueDate,omitempty" json:"defaultValueDate,omitempty"`
	DefaultValueDateTime        *FHIRDateTime    `bson:"defaultValueDateTime,omitempty" json:"defaultValueDateTime,omitempty"`
	DefaultValueDecimal         *float64         `bson:"defaultValueDecimal,omitempty" json:"defaultValueDecimal,omitempty"`
	DefaultValueHumanName       *HumanName       `bson:"defaultValueHumanName,omitempty" json:"defaultValueHumanName,omitempty"`
	DefaultValueId              string           `bson:"defaultValueId,omitempty" json:"defaultValueId,omitempty"`
	DefaultValueIdentifier      *Identifier      `bson:"defaultValueIdentifier,omitempty" json:"defaultValueIdentifier,omitempty"`
	DefaultValueInstant         *FHIRDateTime    `bson:"defaultValueInstant,omitempty" json:"defaultValueInstant,omitempty"`
	DefaultValueInteger         *int32           `bson:"defaultValueInteger,omitempty" json:"defaultValueInteger,omitempty"`
	DefaultValueMarkdown        string           `bson:"defaultValueMarkdown,omitempty" json:"defaultValueMarkdown,omitempty"`
	DefaultValueMeta            *Meta            `bson:"defaultValueMeta,omitempty" json:"defaultValueMeta,omitempty"`
	DefaultValueOid             string           `bson:"defaultValueOid,omitempty" json:"defaultValueOid,omitempty"`
	DefaultValuePeriod          *Period          `bson:"defaultValuePeriod,omitempty" json:"defaultValuePeriod,omitempty"`
	DefaultValuePositiveInt     *uint32          `bson:"defaultValuePositiveInt,omitempty" json:"defaultValuePositiveInt,omitempty"`
	DefaultValueQuantity        *Quantity        `bson:"defaultValueQuantity,omitempty" json:"defaultValueQuantity,omitempty"`
	DefaultValueRange           *Range           `bson:"defaultValueRange,omitempty" json:"defaultValueRange,omitempty"`
	DefaultValueRatio           *Ratio           `bson:"defaultValueRatio,omitempty" json:"defaultValueRatio,omitempty"`
	DefaultValueReference       *Reference       `bson:"defaultValueReference,omitempty" json:"defaultValueReference,omitempty"`
	DefaultValueSampledData     *SampledData     `bson:"defaultValueSampledData,omitempty" json:"defaultValueSampledData,omitempty"`
	DefaultValueSignature       *Signature       `bson:"defaultValueSignature,omitempty" json:"defaultValueSignature,omitempty"`
	DefaultValueString          string           `bson:"defaultValueString,omitempty" json:"defaultValueString,omitempty"`
	DefaultValueTime            *FHIRDateTime    `bson:"defaultValueTime,omitempty" json:"defaultValueTime,omitempty"`
	DefaultValueTiming          *Timing          `bson:"defaultValueTiming,omitempty" json:"defaultValueTiming,omitempty"`
	DefaultValueUnsignedInt     *uint32          `bson:"defaultValueUnsignedInt,omitempty" json:"defaultValueUnsignedInt,omitempty"`
	DefaultValueUri             string           `bson:"defaultValueUri,omitempty" json:"defaultValueUri,omitempty"`
	Element                     string           `bson:"element,omitempty" json:"element,omitempty"`
	ListMode                    string           `bson:"listMode,omitempty" json:"listMode,omitempty"`
	Variable                    string           `bson:"variable,omitempty" json:"variable,omitempty"`
	Condition                   string           `bson:"condition,omitempty" json:"condition,omitempty"`
	Check                       string           `bson:"check,omitempty" json:"check,omitempty"`
}

type StructureMapGroupRuleTargetComponent struct {
	BackboneElement `bson:",inline"`
	Context         string                                          `bson:"context,omitempty" json:"context,omitempty"`
	ContextType     string                                          `bson:"contextType,omitempty" json:"contextType,omitempty"`
	Element         string                                          `bson:"element,omitempty" json:"element,omitempty"`
	Variable        string                                          `bson:"variable,omitempty" json:"variable,omitempty"`
	ListMode        []string                                        `bson:"listMode,omitempty" json:"listMode,omitempty"`
	ListRuleId      string                                          `bson:"listRuleId,omitempty" json:"listRuleId,omitempty"`
	Transform       string                                          `bson:"transform,omitempty" json:"transform,omitempty"`
	Parameter       []StructureMapGroupRuleTargetParameterComponent `bson:"parameter,omitempty" json:"parameter,omitempty"`
}

type StructureMapGroupRuleTargetParameterComponent struct {
	BackboneElement `bson:",inline"`
	ValueId         string   `bson:"valueId,omitempty" json:"valueId,omitempty"`
	ValueString     string   `bson:"valueString,omitempty" json:"valueString,omitempty"`
	ValueBoolean    *bool    `bson:"valueBoolean,omitempty" json:"valueBoolean,omitempty"`
	ValueInteger    *int32   `bson:"valueInteger,omitempty" json:"valueInteger,omitempty"`
	ValueDecimal    *float64 `bson:"valueDecimal,omitempty" json:"valueDecimal,omitempty"`
}

type StructureMapGroupRuleDependentComponent struct {
	BackboneElement `bson:",inline"`
	Name            string   `bson:"name,omitempty" json:"name,omitempty"`
	Variable        []string `bson:"variable,omitempty" json:"variable,omitempty"`
}
