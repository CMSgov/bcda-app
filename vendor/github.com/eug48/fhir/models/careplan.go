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

type CarePlan struct {
	DomainResource `bson:",inline"`
	Identifier     []Identifier                `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Definition     []Reference                 `bson:"definition,omitempty" json:"definition,omitempty"`
	BasedOn        []Reference                 `bson:"basedOn,omitempty" json:"basedOn,omitempty"`
	Replaces       []Reference                 `bson:"replaces,omitempty" json:"replaces,omitempty"`
	PartOf         []Reference                 `bson:"partOf,omitempty" json:"partOf,omitempty"`
	Status         string                      `bson:"status,omitempty" json:"status,omitempty"`
	Intent         string                      `bson:"intent,omitempty" json:"intent,omitempty"`
	Category       []CodeableConcept           `bson:"category,omitempty" json:"category,omitempty"`
	Title          string                      `bson:"title,omitempty" json:"title,omitempty"`
	Description    string                      `bson:"description,omitempty" json:"description,omitempty"`
	Subject        *Reference                  `bson:"subject,omitempty" json:"subject,omitempty"`
	Context        *Reference                  `bson:"context,omitempty" json:"context,omitempty"`
	Period         *Period                     `bson:"period,omitempty" json:"period,omitempty"`
	Author         []Reference                 `bson:"author,omitempty" json:"author,omitempty"`
	CareTeam       []Reference                 `bson:"careTeam,omitempty" json:"careTeam,omitempty"`
	Addresses      []Reference                 `bson:"addresses,omitempty" json:"addresses,omitempty"`
	SupportingInfo []Reference                 `bson:"supportingInfo,omitempty" json:"supportingInfo,omitempty"`
	Goal           []Reference                 `bson:"goal,omitempty" json:"goal,omitempty"`
	Activity       []CarePlanActivityComponent `bson:"activity,omitempty" json:"activity,omitempty"`
	Note           []Annotation                `bson:"note,omitempty" json:"note,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *CarePlan) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "CarePlan"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to CarePlan), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *CarePlan) GetBSON() (interface{}, error) {
	x.ResourceType = "CarePlan"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "carePlan" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type carePlan CarePlan

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *CarePlan) UnmarshalJSON(data []byte) (err error) {
	x2 := carePlan{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = CarePlan(x2)
		return x.checkResourceType()
	}
	return
}

func (x *CarePlan) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "CarePlan"
	} else if x.ResourceType != "CarePlan" {
		return errors.New(fmt.Sprintf("Expected resourceType to be CarePlan, instead received %s", x.ResourceType))
	}
	return nil
}

type CarePlanActivityComponent struct {
	BackboneElement        `bson:",inline"`
	OutcomeCodeableConcept []CodeableConcept                `bson:"outcomeCodeableConcept,omitempty" json:"outcomeCodeableConcept,omitempty"`
	OutcomeReference       []Reference                      `bson:"outcomeReference,omitempty" json:"outcomeReference,omitempty"`
	Progress               []Annotation                     `bson:"progress,omitempty" json:"progress,omitempty"`
	Reference              *Reference                       `bson:"reference,omitempty" json:"reference,omitempty"`
	Detail                 *CarePlanActivityDetailComponent `bson:"detail,omitempty" json:"detail,omitempty"`
}

type CarePlanActivityDetailComponent struct {
	BackboneElement        `bson:",inline"`
	Category               *CodeableConcept  `bson:"category,omitempty" json:"category,omitempty"`
	Definition             *Reference        `bson:"definition,omitempty" json:"definition,omitempty"`
	Code                   *CodeableConcept  `bson:"code,omitempty" json:"code,omitempty"`
	ReasonCode             []CodeableConcept `bson:"reasonCode,omitempty" json:"reasonCode,omitempty"`
	ReasonReference        []Reference       `bson:"reasonReference,omitempty" json:"reasonReference,omitempty"`
	Goal                   []Reference       `bson:"goal,omitempty" json:"goal,omitempty"`
	Status                 string            `bson:"status,omitempty" json:"status,omitempty"`
	StatusReason           string            `bson:"statusReason,omitempty" json:"statusReason,omitempty"`
	Prohibited             *bool             `bson:"prohibited,omitempty" json:"prohibited,omitempty"`
	ScheduledTiming        *Timing           `bson:"scheduledTiming,omitempty" json:"scheduledTiming,omitempty"`
	ScheduledPeriod        *Period           `bson:"scheduledPeriod,omitempty" json:"scheduledPeriod,omitempty"`
	ScheduledString        string            `bson:"scheduledString,omitempty" json:"scheduledString,omitempty"`
	Location               *Reference        `bson:"location,omitempty" json:"location,omitempty"`
	Performer              []Reference       `bson:"performer,omitempty" json:"performer,omitempty"`
	ProductCodeableConcept *CodeableConcept  `bson:"productCodeableConcept,omitempty" json:"productCodeableConcept,omitempty"`
	ProductReference       *Reference        `bson:"productReference,omitempty" json:"productReference,omitempty"`
	DailyAmount            *Quantity         `bson:"dailyAmount,omitempty" json:"dailyAmount,omitempty"`
	Quantity               *Quantity         `bson:"quantity,omitempty" json:"quantity,omitempty"`
	Description            string            `bson:"description,omitempty" json:"description,omitempty"`
}
