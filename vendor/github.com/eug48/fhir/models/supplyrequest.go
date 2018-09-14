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

type SupplyRequest struct {
	DomainResource        `bson:",inline"`
	Identifier            *Identifier                        `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status                string                             `bson:"status,omitempty" json:"status,omitempty"`
	Category              *CodeableConcept                   `bson:"category,omitempty" json:"category,omitempty"`
	Priority              string                             `bson:"priority,omitempty" json:"priority,omitempty"`
	OrderedItem           *SupplyRequestOrderedItemComponent `bson:"orderedItem,omitempty" json:"orderedItem,omitempty"`
	OccurrenceDateTime    *FHIRDateTime                      `bson:"occurrenceDateTime,omitempty" json:"occurrenceDateTime,omitempty"`
	OccurrencePeriod      *Period                            `bson:"occurrencePeriod,omitempty" json:"occurrencePeriod,omitempty"`
	OccurrenceTiming      *Timing                            `bson:"occurrenceTiming,omitempty" json:"occurrenceTiming,omitempty"`
	AuthoredOn            *FHIRDateTime                      `bson:"authoredOn,omitempty" json:"authoredOn,omitempty"`
	Requester             *SupplyRequestRequesterComponent   `bson:"requester,omitempty" json:"requester,omitempty"`
	Supplier              []Reference                        `bson:"supplier,omitempty" json:"supplier,omitempty"`
	ReasonCodeableConcept *CodeableConcept                   `bson:"reasonCodeableConcept,omitempty" json:"reasonCodeableConcept,omitempty"`
	ReasonReference       *Reference                         `bson:"reasonReference,omitempty" json:"reasonReference,omitempty"`
	DeliverFrom           *Reference                         `bson:"deliverFrom,omitempty" json:"deliverFrom,omitempty"`
	DeliverTo             *Reference                         `bson:"deliverTo,omitempty" json:"deliverTo,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *SupplyRequest) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "SupplyRequest"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to SupplyRequest), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *SupplyRequest) GetBSON() (interface{}, error) {
	x.ResourceType = "SupplyRequest"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "supplyRequest" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type supplyRequest SupplyRequest

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *SupplyRequest) UnmarshalJSON(data []byte) (err error) {
	x2 := supplyRequest{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = SupplyRequest(x2)
		return x.checkResourceType()
	}
	return
}

func (x *SupplyRequest) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "SupplyRequest"
	} else if x.ResourceType != "SupplyRequest" {
		return errors.New(fmt.Sprintf("Expected resourceType to be SupplyRequest, instead received %s", x.ResourceType))
	}
	return nil
}

type SupplyRequestOrderedItemComponent struct {
	BackboneElement     `bson:",inline"`
	Quantity            *Quantity        `bson:"quantity,omitempty" json:"quantity,omitempty"`
	ItemCodeableConcept *CodeableConcept `bson:"itemCodeableConcept,omitempty" json:"itemCodeableConcept,omitempty"`
	ItemReference       *Reference       `bson:"itemReference,omitempty" json:"itemReference,omitempty"`
}

type SupplyRequestRequesterComponent struct {
	BackboneElement `bson:",inline"`
	Agent           *Reference `bson:"agent,omitempty" json:"agent,omitempty"`
	OnBehalfOf      *Reference `bson:"onBehalfOf,omitempty" json:"onBehalfOf,omitempty"`
}
