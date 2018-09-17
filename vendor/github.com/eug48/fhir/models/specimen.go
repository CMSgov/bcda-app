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

type Specimen struct {
	DomainResource      `bson:",inline"`
	Identifier          []Identifier                  `bson:"identifier,omitempty" json:"identifier,omitempty"`
	AccessionIdentifier *Identifier                   `bson:"accessionIdentifier,omitempty" json:"accessionIdentifier,omitempty"`
	Status              string                        `bson:"status,omitempty" json:"status,omitempty"`
	Type                *CodeableConcept              `bson:"type,omitempty" json:"type,omitempty"`
	Subject             *Reference                    `bson:"subject,omitempty" json:"subject,omitempty"`
	ReceivedTime        *FHIRDateTime                 `bson:"receivedTime,omitempty" json:"receivedTime,omitempty"`
	Parent              []Reference                   `bson:"parent,omitempty" json:"parent,omitempty"`
	Request             []Reference                   `bson:"request,omitempty" json:"request,omitempty"`
	Collection          *SpecimenCollectionComponent  `bson:"collection,omitempty" json:"collection,omitempty"`
	Processing          []SpecimenProcessingComponent `bson:"processing,omitempty" json:"processing,omitempty"`
	Container           []SpecimenContainerComponent  `bson:"container,omitempty" json:"container,omitempty"`
	Note                []Annotation                  `bson:"note,omitempty" json:"note,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Specimen) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Specimen"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Specimen), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Specimen) GetBSON() (interface{}, error) {
	x.ResourceType = "Specimen"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "specimen" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type specimen Specimen

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Specimen) UnmarshalJSON(data []byte) (err error) {
	x2 := specimen{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Specimen(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Specimen) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Specimen"
	} else if x.ResourceType != "Specimen" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Specimen, instead received %s", x.ResourceType))
	}
	return nil
}

type SpecimenCollectionComponent struct {
	BackboneElement   `bson:",inline"`
	Collector         *Reference       `bson:"collector,omitempty" json:"collector,omitempty"`
	CollectedDateTime *FHIRDateTime    `bson:"collectedDateTime,omitempty" json:"collectedDateTime,omitempty"`
	CollectedPeriod   *Period          `bson:"collectedPeriod,omitempty" json:"collectedPeriod,omitempty"`
	Quantity          *Quantity        `bson:"quantity,omitempty" json:"quantity,omitempty"`
	Method            *CodeableConcept `bson:"method,omitempty" json:"method,omitempty"`
	BodySite          *CodeableConcept `bson:"bodySite,omitempty" json:"bodySite,omitempty"`
}

type SpecimenProcessingComponent struct {
	BackboneElement `bson:",inline"`
	Description     string           `bson:"description,omitempty" json:"description,omitempty"`
	Procedure       *CodeableConcept `bson:"procedure,omitempty" json:"procedure,omitempty"`
	Additive        []Reference      `bson:"additive,omitempty" json:"additive,omitempty"`
	TimeDateTime    *FHIRDateTime    `bson:"timeDateTime,omitempty" json:"timeDateTime,omitempty"`
	TimePeriod      *Period          `bson:"timePeriod,omitempty" json:"timePeriod,omitempty"`
}

type SpecimenContainerComponent struct {
	BackboneElement         `bson:",inline"`
	Identifier              []Identifier     `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Description             string           `bson:"description,omitempty" json:"description,omitempty"`
	Type                    *CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	Capacity                *Quantity        `bson:"capacity,omitempty" json:"capacity,omitempty"`
	SpecimenQuantity        *Quantity        `bson:"specimenQuantity,omitempty" json:"specimenQuantity,omitempty"`
	AdditiveCodeableConcept *CodeableConcept `bson:"additiveCodeableConcept,omitempty" json:"additiveCodeableConcept,omitempty"`
	AdditiveReference       *Reference       `bson:"additiveReference,omitempty" json:"additiveReference,omitempty"`
}
