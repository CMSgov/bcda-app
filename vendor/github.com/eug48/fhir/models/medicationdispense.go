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

type MedicationDispense struct {
	DomainResource               `bson:",inline"`
	Identifier                   []Identifier                             `bson:"identifier,omitempty" json:"identifier,omitempty"`
	PartOf                       []Reference                              `bson:"partOf,omitempty" json:"partOf,omitempty"`
	Status                       string                                   `bson:"status,omitempty" json:"status,omitempty"`
	Category                     *CodeableConcept                         `bson:"category,omitempty" json:"category,omitempty"`
	MedicationCodeableConcept    *CodeableConcept                         `bson:"medicationCodeableConcept,omitempty" json:"medicationCodeableConcept,omitempty"`
	MedicationReference          *Reference                               `bson:"medicationReference,omitempty" json:"medicationReference,omitempty"`
	Subject                      *Reference                               `bson:"subject,omitempty" json:"subject,omitempty"`
	Context                      *Reference                               `bson:"context,omitempty" json:"context,omitempty"`
	SupportingInformation        []Reference                              `bson:"supportingInformation,omitempty" json:"supportingInformation,omitempty"`
	Performer                    []MedicationDispensePerformerComponent   `bson:"performer,omitempty" json:"performer,omitempty"`
	AuthorizingPrescription      []Reference                              `bson:"authorizingPrescription,omitempty" json:"authorizingPrescription,omitempty"`
	Type                         *CodeableConcept                         `bson:"type,omitempty" json:"type,omitempty"`
	Quantity                     *Quantity                                `bson:"quantity,omitempty" json:"quantity,omitempty"`
	DaysSupply                   *Quantity                                `bson:"daysSupply,omitempty" json:"daysSupply,omitempty"`
	WhenPrepared                 *FHIRDateTime                            `bson:"whenPrepared,omitempty" json:"whenPrepared,omitempty"`
	WhenHandedOver               *FHIRDateTime                            `bson:"whenHandedOver,omitempty" json:"whenHandedOver,omitempty"`
	Destination                  *Reference                               `bson:"destination,omitempty" json:"destination,omitempty"`
	Receiver                     []Reference                              `bson:"receiver,omitempty" json:"receiver,omitempty"`
	Note                         []Annotation                             `bson:"note,omitempty" json:"note,omitempty"`
	DosageInstruction            []Dosage                                 `bson:"dosageInstruction,omitempty" json:"dosageInstruction,omitempty"`
	Substitution                 *MedicationDispenseSubstitutionComponent `bson:"substitution,omitempty" json:"substitution,omitempty"`
	DetectedIssue                []Reference                              `bson:"detectedIssue,omitempty" json:"detectedIssue,omitempty"`
	NotDone                      *bool                                    `bson:"notDone,omitempty" json:"notDone,omitempty"`
	NotDoneReasonCodeableConcept *CodeableConcept                         `bson:"notDoneReasonCodeableConcept,omitempty" json:"notDoneReasonCodeableConcept,omitempty"`
	NotDoneReasonReference       *Reference                               `bson:"notDoneReasonReference,omitempty" json:"notDoneReasonReference,omitempty"`
	EventHistory                 []Reference                              `bson:"eventHistory,omitempty" json:"eventHistory,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *MedicationDispense) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "MedicationDispense"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to MedicationDispense), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *MedicationDispense) GetBSON() (interface{}, error) {
	x.ResourceType = "MedicationDispense"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "medicationDispense" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type medicationDispense MedicationDispense

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *MedicationDispense) UnmarshalJSON(data []byte) (err error) {
	x2 := medicationDispense{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = MedicationDispense(x2)
		return x.checkResourceType()
	}
	return
}

func (x *MedicationDispense) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "MedicationDispense"
	} else if x.ResourceType != "MedicationDispense" {
		return errors.New(fmt.Sprintf("Expected resourceType to be MedicationDispense, instead received %s", x.ResourceType))
	}
	return nil
}

type MedicationDispensePerformerComponent struct {
	BackboneElement `bson:",inline"`
	Actor           *Reference `bson:"actor,omitempty" json:"actor,omitempty"`
	OnBehalfOf      *Reference `bson:"onBehalfOf,omitempty" json:"onBehalfOf,omitempty"`
}

type MedicationDispenseSubstitutionComponent struct {
	BackboneElement  `bson:",inline"`
	WasSubstituted   *bool             `bson:"wasSubstituted,omitempty" json:"wasSubstituted,omitempty"`
	Type             *CodeableConcept  `bson:"type,omitempty" json:"type,omitempty"`
	Reason           []CodeableConcept `bson:"reason,omitempty" json:"reason,omitempty"`
	ResponsibleParty []Reference       `bson:"responsibleParty,omitempty" json:"responsibleParty,omitempty"`
}
