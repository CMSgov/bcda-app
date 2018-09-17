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

type Schedule struct {
	DomainResource  `bson:",inline"`
	Identifier      []Identifier      `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Active          *bool             `bson:"active,omitempty" json:"active,omitempty"`
	ServiceCategory *CodeableConcept  `bson:"serviceCategory,omitempty" json:"serviceCategory,omitempty"`
	ServiceType     []CodeableConcept `bson:"serviceType,omitempty" json:"serviceType,omitempty"`
	Specialty       []CodeableConcept `bson:"specialty,omitempty" json:"specialty,omitempty"`
	Actor           []Reference       `bson:"actor,omitempty" json:"actor,omitempty"`
	PlanningHorizon *Period           `bson:"planningHorizon,omitempty" json:"planningHorizon,omitempty"`
	Comment         string            `bson:"comment,omitempty" json:"comment,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Schedule) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Schedule"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Schedule), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Schedule) GetBSON() (interface{}, error) {
	x.ResourceType = "Schedule"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "schedule" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type schedule Schedule

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Schedule) UnmarshalJSON(data []byte) (err error) {
	x2 := schedule{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Schedule(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Schedule) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Schedule"
	} else if x.ResourceType != "Schedule" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Schedule, instead received %s", x.ResourceType))
	}
	return nil
}
