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

type Encounter struct {
	DomainResource   `bson:",inline"`
	Identifier       []Identifier                       `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status           string                             `bson:"status,omitempty" json:"status,omitempty"`
	StatusHistory    []EncounterStatusHistoryComponent  `bson:"statusHistory,omitempty" json:"statusHistory,omitempty"`
	Class            *Coding                            `bson:"class,omitempty" json:"class,omitempty"`
	ClassHistory     []EncounterClassHistoryComponent   `bson:"classHistory,omitempty" json:"classHistory,omitempty"`
	Type             []CodeableConcept                  `bson:"type,omitempty" json:"type,omitempty"`
	Priority         *CodeableConcept                   `bson:"priority,omitempty" json:"priority,omitempty"`
	Subject          *Reference                         `bson:"subject,omitempty" json:"subject,omitempty"`
	EpisodeOfCare    []Reference                        `bson:"episodeOfCare,omitempty" json:"episodeOfCare,omitempty"`
	IncomingReferral []Reference                        `bson:"incomingReferral,omitempty" json:"incomingReferral,omitempty"`
	Participant      []EncounterParticipantComponent    `bson:"participant,omitempty" json:"participant,omitempty"`
	Appointment      *Reference                         `bson:"appointment,omitempty" json:"appointment,omitempty"`
	Period           *Period                            `bson:"period,omitempty" json:"period,omitempty"`
	Length           *Quantity                          `bson:"length,omitempty" json:"length,omitempty"`
	Reason           []CodeableConcept                  `bson:"reason,omitempty" json:"reason,omitempty"`
	Diagnosis        []EncounterDiagnosisComponent      `bson:"diagnosis,omitempty" json:"diagnosis,omitempty"`
	Account          []Reference                        `bson:"account,omitempty" json:"account,omitempty"`
	Hospitalization  *EncounterHospitalizationComponent `bson:"hospitalization,omitempty" json:"hospitalization,omitempty"`
	Location         []EncounterLocationComponent       `bson:"location,omitempty" json:"location,omitempty"`
	ServiceProvider  *Reference                         `bson:"serviceProvider,omitempty" json:"serviceProvider,omitempty"`
	PartOf           *Reference                         `bson:"partOf,omitempty" json:"partOf,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Encounter) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Encounter"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Encounter), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Encounter) GetBSON() (interface{}, error) {
	x.ResourceType = "Encounter"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "encounter" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type encounter Encounter

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Encounter) UnmarshalJSON(data []byte) (err error) {
	x2 := encounter{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Encounter(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Encounter) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Encounter"
	} else if x.ResourceType != "Encounter" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Encounter, instead received %s", x.ResourceType))
	}
	return nil
}

type EncounterStatusHistoryComponent struct {
	BackboneElement `bson:",inline"`
	Status          string  `bson:"status,omitempty" json:"status,omitempty"`
	Period          *Period `bson:"period,omitempty" json:"period,omitempty"`
}

type EncounterClassHistoryComponent struct {
	BackboneElement `bson:",inline"`
	Class           *Coding `bson:"class,omitempty" json:"class,omitempty"`
	Period          *Period `bson:"period,omitempty" json:"period,omitempty"`
}

type EncounterParticipantComponent struct {
	BackboneElement `bson:",inline"`
	Type            []CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	Period          *Period           `bson:"period,omitempty" json:"period,omitempty"`
	Individual      *Reference        `bson:"individual,omitempty" json:"individual,omitempty"`
}

type EncounterDiagnosisComponent struct {
	BackboneElement `bson:",inline"`
	Condition       *Reference       `bson:"condition,omitempty" json:"condition,omitempty"`
	Role            *CodeableConcept `bson:"role,omitempty" json:"role,omitempty"`
	Rank            *uint32          `bson:"rank,omitempty" json:"rank,omitempty"`
}

type EncounterHospitalizationComponent struct {
	BackboneElement        `bson:",inline"`
	PreAdmissionIdentifier *Identifier       `bson:"preAdmissionIdentifier,omitempty" json:"preAdmissionIdentifier,omitempty"`
	Origin                 *Reference        `bson:"origin,omitempty" json:"origin,omitempty"`
	AdmitSource            *CodeableConcept  `bson:"admitSource,omitempty" json:"admitSource,omitempty"`
	ReAdmission            *CodeableConcept  `bson:"reAdmission,omitempty" json:"reAdmission,omitempty"`
	DietPreference         []CodeableConcept `bson:"dietPreference,omitempty" json:"dietPreference,omitempty"`
	SpecialCourtesy        []CodeableConcept `bson:"specialCourtesy,omitempty" json:"specialCourtesy,omitempty"`
	SpecialArrangement     []CodeableConcept `bson:"specialArrangement,omitempty" json:"specialArrangement,omitempty"`
	Destination            *Reference        `bson:"destination,omitempty" json:"destination,omitempty"`
	DischargeDisposition   *CodeableConcept  `bson:"dischargeDisposition,omitempty" json:"dischargeDisposition,omitempty"`
}

type EncounterLocationComponent struct {
	BackboneElement `bson:",inline"`
	Location        *Reference `bson:"location,omitempty" json:"location,omitempty"`
	Status          string     `bson:"status,omitempty" json:"status,omitempty"`
	Period          *Period    `bson:"period,omitempty" json:"period,omitempty"`
}
