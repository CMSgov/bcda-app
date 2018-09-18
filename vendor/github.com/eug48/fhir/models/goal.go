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

type Goal struct {
	DomainResource       `bson:",inline"`
	Identifier           []Identifier         `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status               string               `bson:"status,omitempty" json:"status,omitempty"`
	Category             []CodeableConcept    `bson:"category,omitempty" json:"category,omitempty"`
	Priority             *CodeableConcept     `bson:"priority,omitempty" json:"priority,omitempty"`
	Description          *CodeableConcept     `bson:"description,omitempty" json:"description,omitempty"`
	Subject              *Reference           `bson:"subject,omitempty" json:"subject,omitempty"`
	StartDate            *FHIRDateTime        `bson:"startDate,omitempty" json:"startDate,omitempty"`
	StartCodeableConcept *CodeableConcept     `bson:"startCodeableConcept,omitempty" json:"startCodeableConcept,omitempty"`
	Target               *GoalTargetComponent `bson:"target,omitempty" json:"target,omitempty"`
	StatusDate           *FHIRDateTime        `bson:"statusDate,omitempty" json:"statusDate,omitempty"`
	StatusReason         string               `bson:"statusReason,omitempty" json:"statusReason,omitempty"`
	ExpressedBy          *Reference           `bson:"expressedBy,omitempty" json:"expressedBy,omitempty"`
	Addresses            []Reference          `bson:"addresses,omitempty" json:"addresses,omitempty"`
	Note                 []Annotation         `bson:"note,omitempty" json:"note,omitempty"`
	OutcomeCode          []CodeableConcept    `bson:"outcomeCode,omitempty" json:"outcomeCode,omitempty"`
	OutcomeReference     []Reference          `bson:"outcomeReference,omitempty" json:"outcomeReference,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Goal) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Goal"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Goal), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Goal) GetBSON() (interface{}, error) {
	x.ResourceType = "Goal"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "goal" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type goal Goal

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Goal) UnmarshalJSON(data []byte) (err error) {
	x2 := goal{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Goal(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Goal) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Goal"
	} else if x.ResourceType != "Goal" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Goal, instead received %s", x.ResourceType))
	}
	return nil
}

type GoalTargetComponent struct {
	BackboneElement       `bson:",inline"`
	Measure               *CodeableConcept `bson:"measure,omitempty" json:"measure,omitempty"`
	DetailQuantity        *Quantity        `bson:"detailQuantity,omitempty" json:"detailQuantity,omitempty"`
	DetailRange           *Range           `bson:"detailRange,omitempty" json:"detailRange,omitempty"`
	DetailCodeableConcept *CodeableConcept `bson:"detailCodeableConcept,omitempty" json:"detailCodeableConcept,omitempty"`
	DueDate               *FHIRDateTime    `bson:"dueDate,omitempty" json:"dueDate,omitempty"`
	DueDuration           *Quantity        `bson:"dueDuration,omitempty" json:"dueDuration,omitempty"`
}
