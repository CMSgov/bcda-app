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

type Procedure struct {
	DomainResource     `bson:",inline"`
	Identifier         []Identifier                    `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Definition         []Reference                     `bson:"definition,omitempty" json:"definition,omitempty"`
	BasedOn            []Reference                     `bson:"basedOn,omitempty" json:"basedOn,omitempty"`
	PartOf             []Reference                     `bson:"partOf,omitempty" json:"partOf,omitempty"`
	Status             string                          `bson:"status,omitempty" json:"status,omitempty"`
	NotDone            *bool                           `bson:"notDone,omitempty" json:"notDone,omitempty"`
	NotDoneReason      *CodeableConcept                `bson:"notDoneReason,omitempty" json:"notDoneReason,omitempty"`
	Category           *CodeableConcept                `bson:"category,omitempty" json:"category,omitempty"`
	Code               *CodeableConcept                `bson:"code,omitempty" json:"code,omitempty"`
	Subject            *Reference                      `bson:"subject,omitempty" json:"subject,omitempty"`
	Context            *Reference                      `bson:"context,omitempty" json:"context,omitempty"`
	PerformedDateTime  *FHIRDateTime                   `bson:"performedDateTime,omitempty" json:"performedDateTime,omitempty"`
	PerformedPeriod    *Period                         `bson:"performedPeriod,omitempty" json:"performedPeriod,omitempty"`
	Performer          []ProcedurePerformerComponent   `bson:"performer,omitempty" json:"performer,omitempty"`
	Location           *Reference                      `bson:"location,omitempty" json:"location,omitempty"`
	ReasonCode         []CodeableConcept               `bson:"reasonCode,omitempty" json:"reasonCode,omitempty"`
	ReasonReference    []Reference                     `bson:"reasonReference,omitempty" json:"reasonReference,omitempty"`
	BodySite           []CodeableConcept               `bson:"bodySite,omitempty" json:"bodySite,omitempty"`
	Outcome            *CodeableConcept                `bson:"outcome,omitempty" json:"outcome,omitempty"`
	Report             []Reference                     `bson:"report,omitempty" json:"report,omitempty"`
	Complication       []CodeableConcept               `bson:"complication,omitempty" json:"complication,omitempty"`
	ComplicationDetail []Reference                     `bson:"complicationDetail,omitempty" json:"complicationDetail,omitempty"`
	FollowUp           []CodeableConcept               `bson:"followUp,omitempty" json:"followUp,omitempty"`
	Note               []Annotation                    `bson:"note,omitempty" json:"note,omitempty"`
	FocalDevice        []ProcedureFocalDeviceComponent `bson:"focalDevice,omitempty" json:"focalDevice,omitempty"`
	UsedReference      []Reference                     `bson:"usedReference,omitempty" json:"usedReference,omitempty"`
	UsedCode           []CodeableConcept               `bson:"usedCode,omitempty" json:"usedCode,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Procedure) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Procedure"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Procedure), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Procedure) GetBSON() (interface{}, error) {
	x.ResourceType = "Procedure"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "procedure" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type procedure Procedure

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Procedure) UnmarshalJSON(data []byte) (err error) {
	x2 := procedure{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Procedure(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Procedure) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Procedure"
	} else if x.ResourceType != "Procedure" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Procedure, instead received %s", x.ResourceType))
	}
	return nil
}

type ProcedurePerformerComponent struct {
	BackboneElement `bson:",inline"`
	Role            *CodeableConcept `bson:"role,omitempty" json:"role,omitempty"`
	Actor           *Reference       `bson:"actor,omitempty" json:"actor,omitempty"`
	OnBehalfOf      *Reference       `bson:"onBehalfOf,omitempty" json:"onBehalfOf,omitempty"`
}

type ProcedureFocalDeviceComponent struct {
	BackboneElement `bson:",inline"`
	Action          *CodeableConcept `bson:"action,omitempty" json:"action,omitempty"`
	Manipulated     *Reference       `bson:"manipulated,omitempty" json:"manipulated,omitempty"`
}
