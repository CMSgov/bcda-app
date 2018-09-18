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

type MedicationRequest struct {
	DomainResource            `bson:",inline"`
	Identifier                []Identifier                               `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Definition                []Reference                                `bson:"definition,omitempty" json:"definition,omitempty"`
	BasedOn                   []Reference                                `bson:"basedOn,omitempty" json:"basedOn,omitempty"`
	GroupIdentifier           *Identifier                                `bson:"groupIdentifier,omitempty" json:"groupIdentifier,omitempty"`
	Status                    string                                     `bson:"status,omitempty" json:"status,omitempty"`
	Intent                    string                                     `bson:"intent,omitempty" json:"intent,omitempty"`
	Category                  *CodeableConcept                           `bson:"category,omitempty" json:"category,omitempty"`
	Priority                  string                                     `bson:"priority,omitempty" json:"priority,omitempty"`
	MedicationCodeableConcept *CodeableConcept                           `bson:"medicationCodeableConcept,omitempty" json:"medicationCodeableConcept,omitempty"`
	MedicationReference       *Reference                                 `bson:"medicationReference,omitempty" json:"medicationReference,omitempty"`
	Subject                   *Reference                                 `bson:"subject,omitempty" json:"subject,omitempty"`
	Context                   *Reference                                 `bson:"context,omitempty" json:"context,omitempty"`
	SupportingInformation     []Reference                                `bson:"supportingInformation,omitempty" json:"supportingInformation,omitempty"`
	AuthoredOn                *FHIRDateTime                              `bson:"authoredOn,omitempty" json:"authoredOn,omitempty"`
	Requester                 *MedicationRequestRequesterComponent       `bson:"requester,omitempty" json:"requester,omitempty"`
	Recorder                  *Reference                                 `bson:"recorder,omitempty" json:"recorder,omitempty"`
	ReasonCode                []CodeableConcept                          `bson:"reasonCode,omitempty" json:"reasonCode,omitempty"`
	ReasonReference           []Reference                                `bson:"reasonReference,omitempty" json:"reasonReference,omitempty"`
	Note                      []Annotation                               `bson:"note,omitempty" json:"note,omitempty"`
	DosageInstruction         []Dosage                                   `bson:"dosageInstruction,omitempty" json:"dosageInstruction,omitempty"`
	DispenseRequest           *MedicationRequestDispenseRequestComponent `bson:"dispenseRequest,omitempty" json:"dispenseRequest,omitempty"`
	Substitution              *MedicationRequestSubstitutionComponent    `bson:"substitution,omitempty" json:"substitution,omitempty"`
	PriorPrescription         *Reference                                 `bson:"priorPrescription,omitempty" json:"priorPrescription,omitempty"`
	DetectedIssue             []Reference                                `bson:"detectedIssue,omitempty" json:"detectedIssue,omitempty"`
	EventHistory              []Reference                                `bson:"eventHistory,omitempty" json:"eventHistory,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *MedicationRequest) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "MedicationRequest"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to MedicationRequest), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *MedicationRequest) GetBSON() (interface{}, error) {
	x.ResourceType = "MedicationRequest"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "medicationRequest" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type medicationRequest MedicationRequest

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *MedicationRequest) UnmarshalJSON(data []byte) (err error) {
	x2 := medicationRequest{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = MedicationRequest(x2)
		return x.checkResourceType()
	}
	return
}

func (x *MedicationRequest) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "MedicationRequest"
	} else if x.ResourceType != "MedicationRequest" {
		return errors.New(fmt.Sprintf("Expected resourceType to be MedicationRequest, instead received %s", x.ResourceType))
	}
	return nil
}

type MedicationRequestRequesterComponent struct {
	BackboneElement `bson:",inline"`
	Agent           *Reference `bson:"agent,omitempty" json:"agent,omitempty"`
	OnBehalfOf      *Reference `bson:"onBehalfOf,omitempty" json:"onBehalfOf,omitempty"`
}

type MedicationRequestDispenseRequestComponent struct {
	BackboneElement        `bson:",inline"`
	ValidityPeriod         *Period    `bson:"validityPeriod,omitempty" json:"validityPeriod,omitempty"`
	NumberOfRepeatsAllowed *uint32    `bson:"numberOfRepeatsAllowed,omitempty" json:"numberOfRepeatsAllowed,omitempty"`
	Quantity               *Quantity  `bson:"quantity,omitempty" json:"quantity,omitempty"`
	ExpectedSupplyDuration *Quantity  `bson:"expectedSupplyDuration,omitempty" json:"expectedSupplyDuration,omitempty"`
	Performer              *Reference `bson:"performer,omitempty" json:"performer,omitempty"`
}

type MedicationRequestSubstitutionComponent struct {
	BackboneElement `bson:",inline"`
	Allowed         *bool            `bson:"allowed,omitempty" json:"allowed,omitempty"`
	Reason          *CodeableConcept `bson:"reason,omitempty" json:"reason,omitempty"`
}
