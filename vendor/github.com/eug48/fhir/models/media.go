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

type Media struct {
	DomainResource     `bson:",inline"`
	Identifier         []Identifier      `bson:"identifier,omitempty" json:"identifier,omitempty"`
	BasedOn            []Reference       `bson:"basedOn,omitempty" json:"basedOn,omitempty"`
	Type               string            `bson:"type,omitempty" json:"type,omitempty"`
	Subtype            *CodeableConcept  `bson:"subtype,omitempty" json:"subtype,omitempty"`
	View               *CodeableConcept  `bson:"view,omitempty" json:"view,omitempty"`
	Subject            *Reference        `bson:"subject,omitempty" json:"subject,omitempty"`
	Context            *Reference        `bson:"context,omitempty" json:"context,omitempty"`
	OccurrenceDateTime *FHIRDateTime     `bson:"occurrenceDateTime,omitempty" json:"occurrenceDateTime,omitempty"`
	OccurrencePeriod   *Period           `bson:"occurrencePeriod,omitempty" json:"occurrencePeriod,omitempty"`
	Operator           *Reference        `bson:"operator,omitempty" json:"operator,omitempty"`
	ReasonCode         []CodeableConcept `bson:"reasonCode,omitempty" json:"reasonCode,omitempty"`
	BodySite           *CodeableConcept  `bson:"bodySite,omitempty" json:"bodySite,omitempty"`
	Device             *Reference        `bson:"device,omitempty" json:"device,omitempty"`
	Height             *uint32           `bson:"height,omitempty" json:"height,omitempty"`
	Width              *uint32           `bson:"width,omitempty" json:"width,omitempty"`
	Frames             *uint32           `bson:"frames,omitempty" json:"frames,omitempty"`
	Duration           *uint32           `bson:"duration,omitempty" json:"duration,omitempty"`
	Content            *Attachment       `bson:"content,omitempty" json:"content,omitempty"`
	Note               []Annotation      `bson:"note,omitempty" json:"note,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Media) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Media"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Media), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Media) GetBSON() (interface{}, error) {
	x.ResourceType = "Media"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "media" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type media Media

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Media) UnmarshalJSON(data []byte) (err error) {
	x2 := media{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Media(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Media) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Media"
	} else if x.ResourceType != "Media" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Media, instead received %s", x.ResourceType))
	}
	return nil
}
