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

type EpisodeOfCare struct {
	DomainResource       `bson:",inline"`
	Identifier           []Identifier                          `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status               string                                `bson:"status,omitempty" json:"status,omitempty"`
	StatusHistory        []EpisodeOfCareStatusHistoryComponent `bson:"statusHistory,omitempty" json:"statusHistory,omitempty"`
	Type                 []CodeableConcept                     `bson:"type,omitempty" json:"type,omitempty"`
	Diagnosis            []EpisodeOfCareDiagnosisComponent     `bson:"diagnosis,omitempty" json:"diagnosis,omitempty"`
	Patient              *Reference                            `bson:"patient,omitempty" json:"patient,omitempty"`
	ManagingOrganization *Reference                            `bson:"managingOrganization,omitempty" json:"managingOrganization,omitempty"`
	Period               *Period                               `bson:"period,omitempty" json:"period,omitempty"`
	ReferralRequest      []Reference                           `bson:"referralRequest,omitempty" json:"referralRequest,omitempty"`
	CareManager          *Reference                            `bson:"careManager,omitempty" json:"careManager,omitempty"`
	Team                 []Reference                           `bson:"team,omitempty" json:"team,omitempty"`
	Account              []Reference                           `bson:"account,omitempty" json:"account,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *EpisodeOfCare) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "EpisodeOfCare"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to EpisodeOfCare), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *EpisodeOfCare) GetBSON() (interface{}, error) {
	x.ResourceType = "EpisodeOfCare"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "episodeOfCare" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type episodeOfCare EpisodeOfCare

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *EpisodeOfCare) UnmarshalJSON(data []byte) (err error) {
	x2 := episodeOfCare{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = EpisodeOfCare(x2)
		return x.checkResourceType()
	}
	return
}

func (x *EpisodeOfCare) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "EpisodeOfCare"
	} else if x.ResourceType != "EpisodeOfCare" {
		return errors.New(fmt.Sprintf("Expected resourceType to be EpisodeOfCare, instead received %s", x.ResourceType))
	}
	return nil
}

type EpisodeOfCareStatusHistoryComponent struct {
	BackboneElement `bson:",inline"`
	Status          string  `bson:"status,omitempty" json:"status,omitempty"`
	Period          *Period `bson:"period,omitempty" json:"period,omitempty"`
}

type EpisodeOfCareDiagnosisComponent struct {
	BackboneElement `bson:",inline"`
	Condition       *Reference       `bson:"condition,omitempty" json:"condition,omitempty"`
	Role            *CodeableConcept `bson:"role,omitempty" json:"role,omitempty"`
	Rank            *uint32          `bson:"rank,omitempty" json:"rank,omitempty"`
}
