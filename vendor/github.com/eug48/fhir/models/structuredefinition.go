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

type StructureDefinition struct {
	DomainResource   `bson:",inline"`
	Url              string                                    `bson:"url,omitempty" json:"url,omitempty"`
	Identifier       []Identifier                              `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Version          string                                    `bson:"version,omitempty" json:"version,omitempty"`
	Name             string                                    `bson:"name,omitempty" json:"name,omitempty"`
	Title            string                                    `bson:"title,omitempty" json:"title,omitempty"`
	Status           string                                    `bson:"status,omitempty" json:"status,omitempty"`
	Experimental     *bool                                     `bson:"experimental,omitempty" json:"experimental,omitempty"`
	Date             *FHIRDateTime                             `bson:"date,omitempty" json:"date,omitempty"`
	Publisher        string                                    `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Contact          []ContactDetail                           `bson:"contact,omitempty" json:"contact,omitempty"`
	Description      string                                    `bson:"description,omitempty" json:"description,omitempty"`
	UseContext       []UsageContext                            `bson:"useContext,omitempty" json:"useContext,omitempty"`
	Jurisdiction     []CodeableConcept                         `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	Purpose          string                                    `bson:"purpose,omitempty" json:"purpose,omitempty"`
	Copyright        string                                    `bson:"copyright,omitempty" json:"copyright,omitempty"`
	Keyword          []Coding                                  `bson:"keyword,omitempty" json:"keyword,omitempty"`
	FhirVersion      string                                    `bson:"fhirVersion,omitempty" json:"fhirVersion,omitempty"`
	Mapping          []StructureDefinitionMappingComponent     `bson:"mapping,omitempty" json:"mapping,omitempty"`
	Kind             string                                    `bson:"kind,omitempty" json:"kind,omitempty"`
	Abstract         *bool                                     `bson:"abstract,omitempty" json:"abstract,omitempty"`
	ContextType      string                                    `bson:"contextType,omitempty" json:"contextType,omitempty"`
	Context          []string                                  `bson:"context,omitempty" json:"context,omitempty"`
	ContextInvariant []string                                  `bson:"contextInvariant,omitempty" json:"contextInvariant,omitempty"`
	Type             string                                    `bson:"type,omitempty" json:"type,omitempty"`
	BaseDefinition   string                                    `bson:"baseDefinition,omitempty" json:"baseDefinition,omitempty"`
	Derivation       string                                    `bson:"derivation,omitempty" json:"derivation,omitempty"`
	Snapshot         *StructureDefinitionSnapshotComponent     `bson:"snapshot,omitempty" json:"snapshot,omitempty"`
	Differential     *StructureDefinitionDifferentialComponent `bson:"differential,omitempty" json:"differential,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *StructureDefinition) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "StructureDefinition"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to StructureDefinition), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *StructureDefinition) GetBSON() (interface{}, error) {
	x.ResourceType = "StructureDefinition"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "structureDefinition" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type structureDefinition StructureDefinition

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *StructureDefinition) UnmarshalJSON(data []byte) (err error) {
	x2 := structureDefinition{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = StructureDefinition(x2)
		return x.checkResourceType()
	}
	return
}

func (x *StructureDefinition) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "StructureDefinition"
	} else if x.ResourceType != "StructureDefinition" {
		return errors.New(fmt.Sprintf("Expected resourceType to be StructureDefinition, instead received %s", x.ResourceType))
	}
	return nil
}

type StructureDefinitionMappingComponent struct {
	BackboneElement `bson:",inline"`
	Identity        string `bson:"identity,omitempty" json:"identity,omitempty"`
	Uri             string `bson:"uri,omitempty" json:"uri,omitempty"`
	Name            string `bson:"name,omitempty" json:"name,omitempty"`
	Comment         string `bson:"comment,omitempty" json:"comment,omitempty"`
}

type StructureDefinitionSnapshotComponent struct {
	BackboneElement `bson:",inline"`
	Element         []ElementDefinition `bson:"element,omitempty" json:"element,omitempty"`
}

type StructureDefinitionDifferentialComponent struct {
	BackboneElement `bson:",inline"`
	Element         []ElementDefinition `bson:"element,omitempty" json:"element,omitempty"`
}
