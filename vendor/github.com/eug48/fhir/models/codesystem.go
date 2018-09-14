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

type CodeSystem struct {
	DomainResource   `bson:",inline"`
	Url              string                                 `bson:"url,omitempty" json:"url,omitempty"`
	Identifier       *Identifier                            `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Version          string                                 `bson:"version,omitempty" json:"version,omitempty"`
	Name             string                                 `bson:"name,omitempty" json:"name,omitempty"`
	Title            string                                 `bson:"title,omitempty" json:"title,omitempty"`
	Status           string                                 `bson:"status,omitempty" json:"status,omitempty"`
	Experimental     *bool                                  `bson:"experimental,omitempty" json:"experimental,omitempty"`
	Date             *FHIRDateTime                          `bson:"date,omitempty" json:"date,omitempty"`
	Publisher        string                                 `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Contact          []ContactDetail                        `bson:"contact,omitempty" json:"contact,omitempty"`
	Description      string                                 `bson:"description,omitempty" json:"description,omitempty"`
	UseContext       []UsageContext                         `bson:"useContext,omitempty" json:"useContext,omitempty"`
	Jurisdiction     []CodeableConcept                      `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	Purpose          string                                 `bson:"purpose,omitempty" json:"purpose,omitempty"`
	Copyright        string                                 `bson:"copyright,omitempty" json:"copyright,omitempty"`
	CaseSensitive    *bool                                  `bson:"caseSensitive,omitempty" json:"caseSensitive,omitempty"`
	ValueSet         string                                 `bson:"valueSet,omitempty" json:"valueSet,omitempty"`
	HierarchyMeaning string                                 `bson:"hierarchyMeaning,omitempty" json:"hierarchyMeaning,omitempty"`
	Compositional    *bool                                  `bson:"compositional,omitempty" json:"compositional,omitempty"`
	VersionNeeded    *bool                                  `bson:"versionNeeded,omitempty" json:"versionNeeded,omitempty"`
	Content          string                                 `bson:"content,omitempty" json:"content,omitempty"`
	Count            *uint32                                `bson:"count,omitempty" json:"count,omitempty"`
	Filter           []CodeSystemFilterComponent            `bson:"filter,omitempty" json:"filter,omitempty"`
	Property         []CodeSystemPropertyComponent          `bson:"property,omitempty" json:"property,omitempty"`
	Concept          []CodeSystemConceptDefinitionComponent `bson:"concept,omitempty" json:"concept,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *CodeSystem) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "CodeSystem"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to CodeSystem), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *CodeSystem) GetBSON() (interface{}, error) {
	x.ResourceType = "CodeSystem"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "codeSystem" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type codeSystem CodeSystem

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *CodeSystem) UnmarshalJSON(data []byte) (err error) {
	x2 := codeSystem{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = CodeSystem(x2)
		return x.checkResourceType()
	}
	return
}

func (x *CodeSystem) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "CodeSystem"
	} else if x.ResourceType != "CodeSystem" {
		return errors.New(fmt.Sprintf("Expected resourceType to be CodeSystem, instead received %s", x.ResourceType))
	}
	return nil
}

type CodeSystemFilterComponent struct {
	BackboneElement `bson:",inline"`
	Code            string   `bson:"code,omitempty" json:"code,omitempty"`
	Description     string   `bson:"description,omitempty" json:"description,omitempty"`
	Operator        []string `bson:"operator,omitempty" json:"operator,omitempty"`
	Value           string   `bson:"value,omitempty" json:"value,omitempty"`
}

type CodeSystemPropertyComponent struct {
	BackboneElement `bson:",inline"`
	Code            string `bson:"code,omitempty" json:"code,omitempty"`
	Uri             string `bson:"uri,omitempty" json:"uri,omitempty"`
	Description     string `bson:"description,omitempty" json:"description,omitempty"`
	Type            string `bson:"type,omitempty" json:"type,omitempty"`
}

type CodeSystemConceptDefinitionComponent struct {
	BackboneElement `bson:",inline"`
	Code            string                                            `bson:"code,omitempty" json:"code,omitempty"`
	Display         string                                            `bson:"display,omitempty" json:"display,omitempty"`
	Definition      string                                            `bson:"definition,omitempty" json:"definition,omitempty"`
	Designation     []CodeSystemConceptDefinitionDesignationComponent `bson:"designation,omitempty" json:"designation,omitempty"`
	Property        []CodeSystemConceptPropertyComponent              `bson:"property,omitempty" json:"property,omitempty"`
	Concept         []CodeSystemConceptDefinitionComponent            `bson:"concept,omitempty" json:"concept,omitempty"`
}

type CodeSystemConceptDefinitionDesignationComponent struct {
	BackboneElement `bson:",inline"`
	Language        string  `bson:"language,omitempty" json:"language,omitempty"`
	Use             *Coding `bson:"use,omitempty" json:"use,omitempty"`
	Value           string  `bson:"value,omitempty" json:"value,omitempty"`
}

type CodeSystemConceptPropertyComponent struct {
	BackboneElement `bson:",inline"`
	Code            string        `bson:"code,omitempty" json:"code,omitempty"`
	ValueCode       string        `bson:"valueCode,omitempty" json:"valueCode,omitempty"`
	ValueCoding     *Coding       `bson:"valueCoding,omitempty" json:"valueCoding,omitempty"`
	ValueString     string        `bson:"valueString,omitempty" json:"valueString,omitempty"`
	ValueInteger    *int32        `bson:"valueInteger,omitempty" json:"valueInteger,omitempty"`
	ValueBoolean    *bool         `bson:"valueBoolean,omitempty" json:"valueBoolean,omitempty"`
	ValueDateTime   *FHIRDateTime `bson:"valueDateTime,omitempty" json:"valueDateTime,omitempty"`
}
