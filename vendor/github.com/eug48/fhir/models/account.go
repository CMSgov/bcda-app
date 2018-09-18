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

type Account struct {
	DomainResource `bson:",inline"`
	Identifier     []Identifier                `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status         string                      `bson:"status,omitempty" json:"status,omitempty"`
	Type           *CodeableConcept            `bson:"type,omitempty" json:"type,omitempty"`
	Name           string                      `bson:"name,omitempty" json:"name,omitempty"`
	Subject        *Reference                  `bson:"subject,omitempty" json:"subject,omitempty"`
	Period         *Period                     `bson:"period,omitempty" json:"period,omitempty"`
	Active         *Period                     `bson:"active,omitempty" json:"active,omitempty"`
	Balance        *Quantity                   `bson:"balance,omitempty" json:"balance,omitempty"`
	Coverage       []AccountCoverageComponent  `bson:"coverage,omitempty" json:"coverage,omitempty"`
	Owner          *Reference                  `bson:"owner,omitempty" json:"owner,omitempty"`
	Description    string                      `bson:"description,omitempty" json:"description,omitempty"`
	Guarantor      []AccountGuarantorComponent `bson:"guarantor,omitempty" json:"guarantor,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Account) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Account"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Account), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Account) GetBSON() (interface{}, error) {
	x.ResourceType = "Account"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "account" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type account Account

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Account) UnmarshalJSON(data []byte) (err error) {
	x2 := account{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Account(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Account) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Account"
	} else if x.ResourceType != "Account" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Account, instead received %s", x.ResourceType))
	}
	return nil
}

type AccountCoverageComponent struct {
	BackboneElement `bson:",inline"`
	Coverage        *Reference `bson:"coverage,omitempty" json:"coverage,omitempty"`
	Priority        *uint32    `bson:"priority,omitempty" json:"priority,omitempty"`
}

type AccountGuarantorComponent struct {
	BackboneElement `bson:",inline"`
	Party           *Reference `bson:"party,omitempty" json:"party,omitempty"`
	OnHold          *bool      `bson:"onHold,omitempty" json:"onHold,omitempty"`
	Period          *Period    `bson:"period,omitempty" json:"period,omitempty"`
}
