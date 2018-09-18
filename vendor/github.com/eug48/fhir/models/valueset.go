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

type ValueSet struct {
	DomainResource `bson:",inline"`
	Url            string                      `bson:"url,omitempty" json:"url,omitempty"`
	Identifier     []Identifier                `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Version        string                      `bson:"version,omitempty" json:"version,omitempty"`
	Name           string                      `bson:"name,omitempty" json:"name,omitempty"`
	Title          string                      `bson:"title,omitempty" json:"title,omitempty"`
	Status         string                      `bson:"status,omitempty" json:"status,omitempty"`
	Experimental   *bool                       `bson:"experimental,omitempty" json:"experimental,omitempty"`
	Date           *FHIRDateTime               `bson:"date,omitempty" json:"date,omitempty"`
	Publisher      string                      `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Contact        []ContactDetail             `bson:"contact,omitempty" json:"contact,omitempty"`
	Description    string                      `bson:"description,omitempty" json:"description,omitempty"`
	UseContext     []UsageContext              `bson:"useContext,omitempty" json:"useContext,omitempty"`
	Jurisdiction   []CodeableConcept           `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	Immutable      *bool                       `bson:"immutable,omitempty" json:"immutable,omitempty"`
	Purpose        string                      `bson:"purpose,omitempty" json:"purpose,omitempty"`
	Copyright      string                      `bson:"copyright,omitempty" json:"copyright,omitempty"`
	Extensible     *bool                       `bson:"extensible,omitempty" json:"extensible,omitempty"`
	Compose        *ValueSetComposeComponent   `bson:"compose,omitempty" json:"compose,omitempty"`
	Expansion      *ValueSetExpansionComponent `bson:"expansion,omitempty" json:"expansion,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *ValueSet) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "ValueSet"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to ValueSet), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *ValueSet) GetBSON() (interface{}, error) {
	x.ResourceType = "ValueSet"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "valueSet" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type valueSet ValueSet

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *ValueSet) UnmarshalJSON(data []byte) (err error) {
	x2 := valueSet{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = ValueSet(x2)
		return x.checkResourceType()
	}
	return
}

func (x *ValueSet) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "ValueSet"
	} else if x.ResourceType != "ValueSet" {
		return errors.New(fmt.Sprintf("Expected resourceType to be ValueSet, instead received %s", x.ResourceType))
	}
	return nil
}

type ValueSetComposeComponent struct {
	BackboneElement `bson:",inline"`
	LockedDate      *FHIRDateTime                 `bson:"lockedDate,omitempty" json:"lockedDate,omitempty"`
	Inactive        *bool                         `bson:"inactive,omitempty" json:"inactive,omitempty"`
	Include         []ValueSetConceptSetComponent `bson:"include,omitempty" json:"include,omitempty"`
	Exclude         []ValueSetConceptSetComponent `bson:"exclude,omitempty" json:"exclude,omitempty"`
}

type ValueSetConceptSetComponent struct {
	BackboneElement `bson:",inline"`
	System          string                              `bson:"system,omitempty" json:"system,omitempty"`
	Version         string                              `bson:"version,omitempty" json:"version,omitempty"`
	Concept         []ValueSetConceptReferenceComponent `bson:"concept,omitempty" json:"concept,omitempty"`
	Filter          []ValueSetConceptSetFilterComponent `bson:"filter,omitempty" json:"filter,omitempty"`
	ValueSet        []string                            `bson:"valueSet,omitempty" json:"valueSet,omitempty"`
}

type ValueSetConceptReferenceComponent struct {
	BackboneElement `bson:",inline"`
	Code            string                                         `bson:"code,omitempty" json:"code,omitempty"`
	Display         string                                         `bson:"display,omitempty" json:"display,omitempty"`
	Designation     []ValueSetConceptReferenceDesignationComponent `bson:"designation,omitempty" json:"designation,omitempty"`
}

type ValueSetConceptReferenceDesignationComponent struct {
	BackboneElement `bson:",inline"`
	Language        string  `bson:"language,omitempty" json:"language,omitempty"`
	Use             *Coding `bson:"use,omitempty" json:"use,omitempty"`
	Value           string  `bson:"value,omitempty" json:"value,omitempty"`
}

type ValueSetConceptSetFilterComponent struct {
	BackboneElement `bson:",inline"`
	Property        string `bson:"property,omitempty" json:"property,omitempty"`
	Op              string `bson:"op,omitempty" json:"op,omitempty"`
	Value           string `bson:"value,omitempty" json:"value,omitempty"`
}

type ValueSetExpansionComponent struct {
	BackboneElement `bson:",inline"`
	Identifier      string                                `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Timestamp       *FHIRDateTime                         `bson:"timestamp,omitempty" json:"timestamp,omitempty"`
	Total           *int32                                `bson:"total,omitempty" json:"total,omitempty"`
	Offset          *int32                                `bson:"offset,omitempty" json:"offset,omitempty"`
	Parameter       []ValueSetExpansionParameterComponent `bson:"parameter,omitempty" json:"parameter,omitempty"`
	Contains        []ValueSetExpansionContainsComponent  `bson:"contains,omitempty" json:"contains,omitempty"`
}

type ValueSetExpansionParameterComponent struct {
	BackboneElement `bson:",inline"`
	Name            string   `bson:"name,omitempty" json:"name,omitempty"`
	ValueString     string   `bson:"valueString,omitempty" json:"valueString,omitempty"`
	ValueBoolean    *bool    `bson:"valueBoolean,omitempty" json:"valueBoolean,omitempty"`
	ValueInteger    *int32   `bson:"valueInteger,omitempty" json:"valueInteger,omitempty"`
	ValueDecimal    *float64 `bson:"valueDecimal,omitempty" json:"valueDecimal,omitempty"`
	ValueUri        string   `bson:"valueUri,omitempty" json:"valueUri,omitempty"`
	ValueCode       string   `bson:"valueCode,omitempty" json:"valueCode,omitempty"`
}

type ValueSetExpansionContainsComponent struct {
	BackboneElement `bson:",inline"`
	System          string                                         `bson:"system,omitempty" json:"system,omitempty"`
	Abstract        *bool                                          `bson:"abstract,omitempty" json:"abstract,omitempty"`
	Inactive        *bool                                          `bson:"inactive,omitempty" json:"inactive,omitempty"`
	Version         string                                         `bson:"version,omitempty" json:"version,omitempty"`
	Code            string                                         `bson:"code,omitempty" json:"code,omitempty"`
	Display         string                                         `bson:"display,omitempty" json:"display,omitempty"`
	Designation     []ValueSetConceptReferenceDesignationComponent `bson:"designation,omitempty" json:"designation,omitempty"`
	Contains        []ValueSetExpansionContainsComponent           `bson:"contains,omitempty" json:"contains,omitempty"`
}
