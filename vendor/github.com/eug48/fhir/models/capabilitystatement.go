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

type CapabilityStatement struct {
	DomainResource      `bson:",inline"`
	Url                 string                                      `bson:"url,omitempty" json:"url,omitempty"`
	Version             string                                      `bson:"version,omitempty" json:"version,omitempty"`
	Name                string                                      `bson:"name,omitempty" json:"name,omitempty"`
	Title               string                                      `bson:"title,omitempty" json:"title,omitempty"`
	Status              string                                      `bson:"status,omitempty" json:"status,omitempty"`
	Experimental        *bool                                       `bson:"experimental,omitempty" json:"experimental,omitempty"`
	Date                *FHIRDateTime                               `bson:"date,omitempty" json:"date,omitempty"`
	Publisher           string                                      `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Contact             []ContactDetail                             `bson:"contact,omitempty" json:"contact,omitempty"`
	Description         string                                      `bson:"description,omitempty" json:"description,omitempty"`
	UseContext          []UsageContext                              `bson:"useContext,omitempty" json:"useContext,omitempty"`
	Jurisdiction        []CodeableConcept                           `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	Purpose             string                                      `bson:"purpose,omitempty" json:"purpose,omitempty"`
	Copyright           string                                      `bson:"copyright,omitempty" json:"copyright,omitempty"`
	Kind                string                                      `bson:"kind,omitempty" json:"kind,omitempty"`
	Instantiates        []string                                    `bson:"instantiates,omitempty" json:"instantiates,omitempty"`
	Software            *CapabilityStatementSoftwareComponent       `bson:"software,omitempty" json:"software,omitempty"`
	Implementation      *CapabilityStatementImplementationComponent `bson:"implementation,omitempty" json:"implementation,omitempty"`
	FhirVersion         string                                      `bson:"fhirVersion,omitempty" json:"fhirVersion,omitempty"`
	AcceptUnknown       string                                      `bson:"acceptUnknown,omitempty" json:"acceptUnknown,omitempty"`
	Format              []string                                    `bson:"format,omitempty" json:"format,omitempty"`
	PatchFormat         []string                                    `bson:"patchFormat,omitempty" json:"patchFormat,omitempty"`
	ImplementationGuide []string                                    `bson:"implementationGuide,omitempty" json:"implementationGuide,omitempty"`
	Profile             []Reference                                 `bson:"profile,omitempty" json:"profile,omitempty"`
	Rest                []CapabilityStatementRestComponent          `bson:"rest,omitempty" json:"rest,omitempty"`
	Messaging           []CapabilityStatementMessagingComponent     `bson:"messaging,omitempty" json:"messaging,omitempty"`
	Document            []CapabilityStatementDocumentComponent      `bson:"document,omitempty" json:"document,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *CapabilityStatement) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "CapabilityStatement"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to CapabilityStatement), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *CapabilityStatement) GetBSON() (interface{}, error) {
	x.ResourceType = "CapabilityStatement"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "capabilityStatement" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type capabilityStatement CapabilityStatement

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *CapabilityStatement) UnmarshalJSON(data []byte) (err error) {
	x2 := capabilityStatement{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = CapabilityStatement(x2)
		return x.checkResourceType()
	}
	return
}

func (x *CapabilityStatement) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "CapabilityStatement"
	} else if x.ResourceType != "CapabilityStatement" {
		return errors.New(fmt.Sprintf("Expected resourceType to be CapabilityStatement, instead received %s", x.ResourceType))
	}
	return nil
}

type CapabilityStatementSoftwareComponent struct {
	BackboneElement `bson:",inline"`
	Name            string        `bson:"name,omitempty" json:"name,omitempty"`
	Version         string        `bson:"version,omitempty" json:"version,omitempty"`
	ReleaseDate     *FHIRDateTime `bson:"releaseDate,omitempty" json:"releaseDate,omitempty"`
}

type CapabilityStatementImplementationComponent struct {
	BackboneElement `bson:",inline"`
	Description     string `bson:"description,omitempty" json:"description,omitempty"`
	Url             string `bson:"url,omitempty" json:"url,omitempty"`
}

type CapabilityStatementRestComponent struct {
	BackboneElement `bson:",inline"`
	Mode            string                                                `bson:"mode,omitempty" json:"mode,omitempty"`
	Documentation   string                                                `bson:"documentation,omitempty" json:"documentation,omitempty"`
	Security        *CapabilityStatementRestSecurityComponent             `bson:"security,omitempty" json:"security,omitempty"`
	Resource        []CapabilityStatementRestResourceComponent            `bson:"resource,omitempty" json:"resource,omitempty"`
	Interaction     []CapabilityStatementSystemInteractionComponent       `bson:"interaction,omitempty" json:"interaction,omitempty"`
	SearchParam     []CapabilityStatementRestResourceSearchParamComponent `bson:"searchParam,omitempty" json:"searchParam,omitempty"`
	Operation       []CapabilityStatementRestOperationComponent           `bson:"operation,omitempty" json:"operation,omitempty"`
	Compartment     []string                                              `bson:"compartment,omitempty" json:"compartment,omitempty"`
}

type CapabilityStatementRestSecurityComponent struct {
	BackboneElement `bson:",inline"`
	Cors            *bool                                                 `bson:"cors,omitempty" json:"cors,omitempty"`
	Service         []CodeableConcept                                     `bson:"service,omitempty" json:"service,omitempty"`
	Description     string                                                `bson:"description,omitempty" json:"description,omitempty"`
	Certificate     []CapabilityStatementRestSecurityCertificateComponent `bson:"certificate,omitempty" json:"certificate,omitempty"`
}

type CapabilityStatementRestSecurityCertificateComponent struct {
	BackboneElement `bson:",inline"`
	Type            string `bson:"type,omitempty" json:"type,omitempty"`
	Blob            string `bson:"blob,omitempty" json:"blob,omitempty"`
}

type CapabilityStatementRestResourceComponent struct {
	BackboneElement   `bson:",inline"`
	Type              string                                                `bson:"type,omitempty" json:"type,omitempty"`
	Profile           *Reference                                            `bson:"profile,omitempty" json:"profile,omitempty"`
	Documentation     string                                                `bson:"documentation,omitempty" json:"documentation,omitempty"`
	Interaction       []CapabilityStatementResourceInteractionComponent     `bson:"interaction,omitempty" json:"interaction,omitempty"`
	Versioning        string                                                `bson:"versioning,omitempty" json:"versioning,omitempty"`
	ReadHistory       *bool                                                 `bson:"readHistory,omitempty" json:"readHistory,omitempty"`
	UpdateCreate      *bool                                                 `bson:"updateCreate,omitempty" json:"updateCreate,omitempty"`
	ConditionalCreate *bool                                                 `bson:"conditionalCreate,omitempty" json:"conditionalCreate,omitempty"`
	ConditionalRead   string                                                `bson:"conditionalRead,omitempty" json:"conditionalRead,omitempty"`
	ConditionalUpdate *bool                                                 `bson:"conditionalUpdate,omitempty" json:"conditionalUpdate,omitempty"`
	ConditionalDelete string                                                `bson:"conditionalDelete,omitempty" json:"conditionalDelete,omitempty"`
	ReferencePolicy   []string                                              `bson:"referencePolicy,omitempty" json:"referencePolicy,omitempty"`
	SearchInclude     []string                                              `bson:"searchInclude,omitempty" json:"searchInclude,omitempty"`
	SearchRevInclude  []string                                              `bson:"searchRevInclude,omitempty" json:"searchRevInclude,omitempty"`
	SearchParam       []CapabilityStatementRestResourceSearchParamComponent `bson:"searchParam,omitempty" json:"searchParam,omitempty"`
}

type CapabilityStatementResourceInteractionComponent struct {
	BackboneElement `bson:",inline"`
	Code            string `bson:"code,omitempty" json:"code,omitempty"`
	Documentation   string `bson:"documentation,omitempty" json:"documentation,omitempty"`
}

type CapabilityStatementRestResourceSearchParamComponent struct {
	BackboneElement `bson:",inline"`
	Name            string `bson:"name,omitempty" json:"name,omitempty"`
	Definition      string `bson:"definition,omitempty" json:"definition,omitempty"`
	Type            string `bson:"type,omitempty" json:"type,omitempty"`
	Documentation   string `bson:"documentation,omitempty" json:"documentation,omitempty"`
}

type CapabilityStatementSystemInteractionComponent struct {
	BackboneElement `bson:",inline"`
	Code            string `bson:"code,omitempty" json:"code,omitempty"`
	Documentation   string `bson:"documentation,omitempty" json:"documentation,omitempty"`
}

type CapabilityStatementRestOperationComponent struct {
	BackboneElement `bson:",inline"`
	Name            string     `bson:"name,omitempty" json:"name,omitempty"`
	Definition      *Reference `bson:"definition,omitempty" json:"definition,omitempty"`
}

type CapabilityStatementMessagingComponent struct {
	BackboneElement  `bson:",inline"`
	Endpoint         []CapabilityStatementMessagingEndpointComponent         `bson:"endpoint,omitempty" json:"endpoint,omitempty"`
	ReliableCache    *uint32                                                 `bson:"reliableCache,omitempty" json:"reliableCache,omitempty"`
	Documentation    string                                                  `bson:"documentation,omitempty" json:"documentation,omitempty"`
	SupportedMessage []CapabilityStatementMessagingSupportedMessageComponent `bson:"supportedMessage,omitempty" json:"supportedMessage,omitempty"`
	Event            []CapabilityStatementMessagingEventComponent            `bson:"event,omitempty" json:"event,omitempty"`
}

type CapabilityStatementMessagingEndpointComponent struct {
	BackboneElement `bson:",inline"`
	Protocol        *Coding `bson:"protocol,omitempty" json:"protocol,omitempty"`
	Address         string  `bson:"address,omitempty" json:"address,omitempty"`
}

type CapabilityStatementMessagingSupportedMessageComponent struct {
	BackboneElement `bson:",inline"`
	Mode            string     `bson:"mode,omitempty" json:"mode,omitempty"`
	Definition      *Reference `bson:"definition,omitempty" json:"definition,omitempty"`
}

type CapabilityStatementMessagingEventComponent struct {
	BackboneElement `bson:",inline"`
	Code            *Coding    `bson:"code,omitempty" json:"code,omitempty"`
	Category        string     `bson:"category,omitempty" json:"category,omitempty"`
	Mode            string     `bson:"mode,omitempty" json:"mode,omitempty"`
	Focus           string     `bson:"focus,omitempty" json:"focus,omitempty"`
	Request         *Reference `bson:"request,omitempty" json:"request,omitempty"`
	Response        *Reference `bson:"response,omitempty" json:"response,omitempty"`
	Documentation   string     `bson:"documentation,omitempty" json:"documentation,omitempty"`
}

type CapabilityStatementDocumentComponent struct {
	BackboneElement `bson:",inline"`
	Mode            string     `bson:"mode,omitempty" json:"mode,omitempty"`
	Documentation   string     `bson:"documentation,omitempty" json:"documentation,omitempty"`
	Profile         *Reference `bson:"profile,omitempty" json:"profile,omitempty"`
}
