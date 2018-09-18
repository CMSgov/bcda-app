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

type Appointment struct {
	DomainResource        `bson:",inline"`
	Identifier            []Identifier                      `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status                string                            `bson:"status,omitempty" json:"status,omitempty"`
	ServiceCategory       *CodeableConcept                  `bson:"serviceCategory,omitempty" json:"serviceCategory,omitempty"`
	ServiceType           []CodeableConcept                 `bson:"serviceType,omitempty" json:"serviceType,omitempty"`
	Specialty             []CodeableConcept                 `bson:"specialty,omitempty" json:"specialty,omitempty"`
	AppointmentType       *CodeableConcept                  `bson:"appointmentType,omitempty" json:"appointmentType,omitempty"`
	Reason                []CodeableConcept                 `bson:"reason,omitempty" json:"reason,omitempty"`
	Indication            []Reference                       `bson:"indication,omitempty" json:"indication,omitempty"`
	Priority              *uint32                           `bson:"priority,omitempty" json:"priority,omitempty"`
	Description           string                            `bson:"description,omitempty" json:"description,omitempty"`
	SupportingInformation []Reference                       `bson:"supportingInformation,omitempty" json:"supportingInformation,omitempty"`
	Start                 *FHIRDateTime                     `bson:"start,omitempty" json:"start,omitempty"`
	End                   *FHIRDateTime                     `bson:"end,omitempty" json:"end,omitempty"`
	MinutesDuration       *uint32                           `bson:"minutesDuration,omitempty" json:"minutesDuration,omitempty"`
	Slot                  []Reference                       `bson:"slot,omitempty" json:"slot,omitempty"`
	Created               *FHIRDateTime                     `bson:"created,omitempty" json:"created,omitempty"`
	Comment               string                            `bson:"comment,omitempty" json:"comment,omitempty"`
	IncomingReferral      []Reference                       `bson:"incomingReferral,omitempty" json:"incomingReferral,omitempty"`
	Participant           []AppointmentParticipantComponent `bson:"participant,omitempty" json:"participant,omitempty"`
	RequestedPeriod       []Period                          `bson:"requestedPeriod,omitempty" json:"requestedPeriod,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Appointment) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Appointment"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Appointment), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Appointment) GetBSON() (interface{}, error) {
	x.ResourceType = "Appointment"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "appointment" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type appointment Appointment

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Appointment) UnmarshalJSON(data []byte) (err error) {
	x2 := appointment{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Appointment(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Appointment) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Appointment"
	} else if x.ResourceType != "Appointment" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Appointment, instead received %s", x.ResourceType))
	}
	return nil
}

type AppointmentParticipantComponent struct {
	BackboneElement `bson:",inline"`
	Type            []CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	Actor           *Reference        `bson:"actor,omitempty" json:"actor,omitempty"`
	Required        string            `bson:"required,omitempty" json:"required,omitempty"`
	Status          string            `bson:"status,omitempty" json:"status,omitempty"`
}
