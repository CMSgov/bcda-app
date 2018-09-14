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

type ChargeItem struct {
	DomainResource         `bson:",inline"`
	Identifier             *Identifier                      `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Definition             []string                         `bson:"definition,omitempty" json:"definition,omitempty"`
	Status                 string                           `bson:"status,omitempty" json:"status,omitempty"`
	PartOf                 []Reference                      `bson:"partOf,omitempty" json:"partOf,omitempty"`
	Code                   *CodeableConcept                 `bson:"code,omitempty" json:"code,omitempty"`
	Subject                *Reference                       `bson:"subject,omitempty" json:"subject,omitempty"`
	Context                *Reference                       `bson:"context,omitempty" json:"context,omitempty"`
	OccurrenceDateTime     *FHIRDateTime                    `bson:"occurrenceDateTime,omitempty" json:"occurrenceDateTime,omitempty"`
	OccurrencePeriod       *Period                          `bson:"occurrencePeriod,omitempty" json:"occurrencePeriod,omitempty"`
	OccurrenceTiming       *Timing                          `bson:"occurrenceTiming,omitempty" json:"occurrenceTiming,omitempty"`
	Participant            []ChargeItemParticipantComponent `bson:"participant,omitempty" json:"participant,omitempty"`
	PerformingOrganization *Reference                       `bson:"performingOrganization,omitempty" json:"performingOrganization,omitempty"`
	RequestingOrganization *Reference                       `bson:"requestingOrganization,omitempty" json:"requestingOrganization,omitempty"`
	Quantity               *Quantity                        `bson:"quantity,omitempty" json:"quantity,omitempty"`
	Bodysite               []CodeableConcept                `bson:"bodysite,omitempty" json:"bodysite,omitempty"`
	FactorOverride         *float64                         `bson:"factorOverride,omitempty" json:"factorOverride,omitempty"`
	PriceOverride          *Quantity                        `bson:"priceOverride,omitempty" json:"priceOverride,omitempty"`
	OverrideReason         string                           `bson:"overrideReason,omitempty" json:"overrideReason,omitempty"`
	Enterer                *Reference                       `bson:"enterer,omitempty" json:"enterer,omitempty"`
	EnteredDate            *FHIRDateTime                    `bson:"enteredDate,omitempty" json:"enteredDate,omitempty"`
	Reason                 []CodeableConcept                `bson:"reason,omitempty" json:"reason,omitempty"`
	Service                []Reference                      `bson:"service,omitempty" json:"service,omitempty"`
	Account                []Reference                      `bson:"account,omitempty" json:"account,omitempty"`
	Note                   []Annotation                     `bson:"note,omitempty" json:"note,omitempty"`
	SupportingInformation  []Reference                      `bson:"supportingInformation,omitempty" json:"supportingInformation,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *ChargeItem) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "ChargeItem"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to ChargeItem), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *ChargeItem) GetBSON() (interface{}, error) {
	x.ResourceType = "ChargeItem"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "chargeItem" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type chargeItem ChargeItem

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *ChargeItem) UnmarshalJSON(data []byte) (err error) {
	x2 := chargeItem{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = ChargeItem(x2)
		return x.checkResourceType()
	}
	return
}

func (x *ChargeItem) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "ChargeItem"
	} else if x.ResourceType != "ChargeItem" {
		return errors.New(fmt.Sprintf("Expected resourceType to be ChargeItem, instead received %s", x.ResourceType))
	}
	return nil
}

type ChargeItemParticipantComponent struct {
	BackboneElement `bson:",inline"`
	Role            *CodeableConcept `bson:"role,omitempty" json:"role,omitempty"`
	Actor           *Reference       `bson:"actor,omitempty" json:"actor,omitempty"`
}
