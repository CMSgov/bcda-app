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

type OperationDefinition struct {
	DomainResource `bson:",inline"`
	Url            string                                  `bson:"url,omitempty" json:"url,omitempty"`
	Version        string                                  `bson:"version,omitempty" json:"version,omitempty"`
	Name           string                                  `bson:"name,omitempty" json:"name,omitempty"`
	Status         string                                  `bson:"status,omitempty" json:"status,omitempty"`
	Kind           string                                  `bson:"kind,omitempty" json:"kind,omitempty"`
	Experimental   *bool                                   `bson:"experimental,omitempty" json:"experimental,omitempty"`
	Date           *FHIRDateTime                           `bson:"date,omitempty" json:"date,omitempty"`
	Publisher      string                                  `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Contact        []ContactDetail                         `bson:"contact,omitempty" json:"contact,omitempty"`
	Description    string                                  `bson:"description,omitempty" json:"description,omitempty"`
	UseContext     []UsageContext                          `bson:"useContext,omitempty" json:"useContext,omitempty"`
	Jurisdiction   []CodeableConcept                       `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	Purpose        string                                  `bson:"purpose,omitempty" json:"purpose,omitempty"`
	Idempotent     *bool                                   `bson:"idempotent,omitempty" json:"idempotent,omitempty"`
	Code           string                                  `bson:"code,omitempty" json:"code,omitempty"`
	Comment        string                                  `bson:"comment,omitempty" json:"comment,omitempty"`
	Base           *Reference                              `bson:"base,omitempty" json:"base,omitempty"`
	Resource       []string                                `bson:"resource,omitempty" json:"resource,omitempty"`
	System         *bool                                   `bson:"system,omitempty" json:"system,omitempty"`
	Type           *bool                                   `bson:"type,omitempty" json:"type,omitempty"`
	Instance       *bool                                   `bson:"instance,omitempty" json:"instance,omitempty"`
	Parameter      []OperationDefinitionParameterComponent `bson:"parameter,omitempty" json:"parameter,omitempty"`
	Overload       []OperationDefinitionOverloadComponent  `bson:"overload,omitempty" json:"overload,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *OperationDefinition) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "OperationDefinition"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to OperationDefinition), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *OperationDefinition) GetBSON() (interface{}, error) {
	x.ResourceType = "OperationDefinition"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "operationDefinition" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type operationDefinition OperationDefinition

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *OperationDefinition) UnmarshalJSON(data []byte) (err error) {
	x2 := operationDefinition{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = OperationDefinition(x2)
		return x.checkResourceType()
	}
	return
}

func (x *OperationDefinition) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "OperationDefinition"
	} else if x.ResourceType != "OperationDefinition" {
		return errors.New(fmt.Sprintf("Expected resourceType to be OperationDefinition, instead received %s", x.ResourceType))
	}
	return nil
}

type OperationDefinitionParameterComponent struct {
	BackboneElement `bson:",inline"`
	Name            string                                        `bson:"name,omitempty" json:"name,omitempty"`
	Use             string                                        `bson:"use,omitempty" json:"use,omitempty"`
	Min             *int32                                        `bson:"min,omitempty" json:"min,omitempty"`
	Max             string                                        `bson:"max,omitempty" json:"max,omitempty"`
	Documentation   string                                        `bson:"documentation,omitempty" json:"documentation,omitempty"`
	Type            string                                        `bson:"type,omitempty" json:"type,omitempty"`
	SearchType      string                                        `bson:"searchType,omitempty" json:"searchType,omitempty"`
	Profile         *Reference                                    `bson:"profile,omitempty" json:"profile,omitempty"`
	Binding         *OperationDefinitionParameterBindingComponent `bson:"binding,omitempty" json:"binding,omitempty"`
	Part            []OperationDefinitionParameterComponent       `bson:"part,omitempty" json:"part,omitempty"`
}

type OperationDefinitionParameterBindingComponent struct {
	BackboneElement   `bson:",inline"`
	Strength          string     `bson:"strength,omitempty" json:"strength,omitempty"`
	ValueSetUri       string     `bson:"valueSetUri,omitempty" json:"valueSetUri,omitempty"`
	ValueSetReference *Reference `bson:"valueSetReference,omitempty" json:"valueSetReference,omitempty"`
}

type OperationDefinitionOverloadComponent struct {
	BackboneElement `bson:",inline"`
	ParameterName   []string `bson:"parameterName,omitempty" json:"parameterName,omitempty"`
	Comment         string   `bson:"comment,omitempty" json:"comment,omitempty"`
}
