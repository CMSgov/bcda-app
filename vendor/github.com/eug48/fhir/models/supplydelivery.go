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

type SupplyDelivery struct {
	DomainResource     `bson:",inline"`
	Identifier         *Identifier                          `bson:"identifier,omitempty" json:"identifier,omitempty"`
	BasedOn            []Reference                          `bson:"basedOn,omitempty" json:"basedOn,omitempty"`
	PartOf             []Reference                          `bson:"partOf,omitempty" json:"partOf,omitempty"`
	Status             string                               `bson:"status,omitempty" json:"status,omitempty"`
	Patient            *Reference                           `bson:"patient,omitempty" json:"patient,omitempty"`
	Type               *CodeableConcept                     `bson:"type,omitempty" json:"type,omitempty"`
	SuppliedItem       *SupplyDeliverySuppliedItemComponent `bson:"suppliedItem,omitempty" json:"suppliedItem,omitempty"`
	OccurrenceDateTime *FHIRDateTime                        `bson:"occurrenceDateTime,omitempty" json:"occurrenceDateTime,omitempty"`
	OccurrencePeriod   *Period                              `bson:"occurrencePeriod,omitempty" json:"occurrencePeriod,omitempty"`
	OccurrenceTiming   *Timing                              `bson:"occurrenceTiming,omitempty" json:"occurrenceTiming,omitempty"`
	Supplier           *Reference                           `bson:"supplier,omitempty" json:"supplier,omitempty"`
	Destination        *Reference                           `bson:"destination,omitempty" json:"destination,omitempty"`
	Receiver           []Reference                          `bson:"receiver,omitempty" json:"receiver,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *SupplyDelivery) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "SupplyDelivery"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to SupplyDelivery), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *SupplyDelivery) GetBSON() (interface{}, error) {
	x.ResourceType = "SupplyDelivery"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "supplyDelivery" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type supplyDelivery SupplyDelivery

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *SupplyDelivery) UnmarshalJSON(data []byte) (err error) {
	x2 := supplyDelivery{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = SupplyDelivery(x2)
		return x.checkResourceType()
	}
	return
}

func (x *SupplyDelivery) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "SupplyDelivery"
	} else if x.ResourceType != "SupplyDelivery" {
		return errors.New(fmt.Sprintf("Expected resourceType to be SupplyDelivery, instead received %s", x.ResourceType))
	}
	return nil
}

type SupplyDeliverySuppliedItemComponent struct {
	BackboneElement     `bson:",inline"`
	Quantity            *Quantity        `bson:"quantity,omitempty" json:"quantity,omitempty"`
	ItemCodeableConcept *CodeableConcept `bson:"itemCodeableConcept,omitempty" json:"itemCodeableConcept,omitempty"`
	ItemReference       *Reference       `bson:"itemReference,omitempty" json:"itemReference,omitempty"`
}
