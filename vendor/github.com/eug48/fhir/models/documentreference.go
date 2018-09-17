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

type DocumentReference struct {
	DomainResource   `bson:",inline"`
	MasterIdentifier *Identifier                           `bson:"masterIdentifier,omitempty" json:"masterIdentifier,omitempty"`
	Identifier       []Identifier                          `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status           string                                `bson:"status,omitempty" json:"status,omitempty"`
	DocStatus        string                                `bson:"docStatus,omitempty" json:"docStatus,omitempty"`
	Type             *CodeableConcept                      `bson:"type,omitempty" json:"type,omitempty"`
	Class            *CodeableConcept                      `bson:"class,omitempty" json:"class,omitempty"`
	Subject          *Reference                            `bson:"subject,omitempty" json:"subject,omitempty"`
	Created          *FHIRDateTime                         `bson:"created,omitempty" json:"created,omitempty"`
	Indexed          *FHIRDateTime                         `bson:"indexed,omitempty" json:"indexed,omitempty"`
	Author           []Reference                           `bson:"author,omitempty" json:"author,omitempty"`
	Authenticator    *Reference                            `bson:"authenticator,omitempty" json:"authenticator,omitempty"`
	Custodian        *Reference                            `bson:"custodian,omitempty" json:"custodian,omitempty"`
	RelatesTo        []DocumentReferenceRelatesToComponent `bson:"relatesTo,omitempty" json:"relatesTo,omitempty"`
	Description      string                                `bson:"description,omitempty" json:"description,omitempty"`
	SecurityLabel    []CodeableConcept                     `bson:"securityLabel,omitempty" json:"securityLabel,omitempty"`
	Content          []DocumentReferenceContentComponent   `bson:"content,omitempty" json:"content,omitempty"`
	Context          *DocumentReferenceContextComponent    `bson:"context,omitempty" json:"context,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *DocumentReference) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "DocumentReference"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to DocumentReference), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *DocumentReference) GetBSON() (interface{}, error) {
	x.ResourceType = "DocumentReference"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "documentReference" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type documentReference DocumentReference

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *DocumentReference) UnmarshalJSON(data []byte) (err error) {
	x2 := documentReference{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = DocumentReference(x2)
		return x.checkResourceType()
	}
	return
}

func (x *DocumentReference) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "DocumentReference"
	} else if x.ResourceType != "DocumentReference" {
		return errors.New(fmt.Sprintf("Expected resourceType to be DocumentReference, instead received %s", x.ResourceType))
	}
	return nil
}

type DocumentReferenceRelatesToComponent struct {
	BackboneElement `bson:",inline"`
	Code            string     `bson:"code,omitempty" json:"code,omitempty"`
	Target          *Reference `bson:"target,omitempty" json:"target,omitempty"`
}

type DocumentReferenceContentComponent struct {
	BackboneElement `bson:",inline"`
	Attachment      *Attachment `bson:"attachment,omitempty" json:"attachment,omitempty"`
	Format          *Coding     `bson:"format,omitempty" json:"format,omitempty"`
}

type DocumentReferenceContextComponent struct {
	BackboneElement   `bson:",inline"`
	Encounter         *Reference                                 `bson:"encounter,omitempty" json:"encounter,omitempty"`
	Event             []CodeableConcept                          `bson:"event,omitempty" json:"event,omitempty"`
	Period            *Period                                    `bson:"period,omitempty" json:"period,omitempty"`
	FacilityType      *CodeableConcept                           `bson:"facilityType,omitempty" json:"facilityType,omitempty"`
	PracticeSetting   *CodeableConcept                           `bson:"practiceSetting,omitempty" json:"practiceSetting,omitempty"`
	SourcePatientInfo *Reference                                 `bson:"sourcePatientInfo,omitempty" json:"sourcePatientInfo,omitempty"`
	Related           []DocumentReferenceContextRelatedComponent `bson:"related,omitempty" json:"related,omitempty"`
}

type DocumentReferenceContextRelatedComponent struct {
	BackboneElement `bson:",inline"`
	Identifier      *Identifier `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Ref             *Reference  `bson:"ref,omitempty" json:"ref,omitempty"`
}
