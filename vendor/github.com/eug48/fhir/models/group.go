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

type Group struct {
	DomainResource `bson:",inline"`
	Identifier     []Identifier                   `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Active         *bool                          `bson:"active,omitempty" json:"active,omitempty"`
	Type           string                         `bson:"type,omitempty" json:"type,omitempty"`
	Actual         *bool                          `bson:"actual,omitempty" json:"actual,omitempty"`
	Code           *CodeableConcept               `bson:"code,omitempty" json:"code,omitempty"`
	Name           string                         `bson:"name,omitempty" json:"name,omitempty"`
	Quantity       *uint32                        `bson:"quantity,omitempty" json:"quantity,omitempty"`
	Characteristic []GroupCharacteristicComponent `bson:"characteristic,omitempty" json:"characteristic,omitempty"`
	Member         []GroupMemberComponent         `bson:"member,omitempty" json:"member,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Group) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Group"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Group), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Group) GetBSON() (interface{}, error) {
	x.ResourceType = "Group"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "group" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type group Group

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Group) UnmarshalJSON(data []byte) (err error) {
	x2 := group{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Group(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Group) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Group"
	} else if x.ResourceType != "Group" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Group, instead received %s", x.ResourceType))
	}
	return nil
}

type GroupCharacteristicComponent struct {
	BackboneElement      `bson:",inline"`
	Code                 *CodeableConcept `bson:"code,omitempty" json:"code,omitempty"`
	ValueCodeableConcept *CodeableConcept `bson:"valueCodeableConcept,omitempty" json:"valueCodeableConcept,omitempty"`
	ValueBoolean         *bool            `bson:"valueBoolean,omitempty" json:"valueBoolean,omitempty"`
	ValueQuantity        *Quantity        `bson:"valueQuantity,omitempty" json:"valueQuantity,omitempty"`
	ValueRange           *Range           `bson:"valueRange,omitempty" json:"valueRange,omitempty"`
	Exclude              *bool            `bson:"exclude,omitempty" json:"exclude,omitempty"`
	Period               *Period          `bson:"period,omitempty" json:"period,omitempty"`
}

type GroupMemberComponent struct {
	BackboneElement `bson:",inline"`
	Entity          *Reference `bson:"entity,omitempty" json:"entity,omitempty"`
	Period          *Period    `bson:"period,omitempty" json:"period,omitempty"`
	Inactive        *bool      `bson:"inactive,omitempty" json:"inactive,omitempty"`
}
