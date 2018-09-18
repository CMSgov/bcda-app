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

type Provenance struct {
	DomainResource `bson:",inline"`
	Target         []Reference                 `bson:"target,omitempty" json:"target,omitempty"`
	Period         *Period                     `bson:"period,omitempty" json:"period,omitempty"`
	Recorded       *FHIRDateTime               `bson:"recorded,omitempty" json:"recorded,omitempty"`
	Policy         []string                    `bson:"policy,omitempty" json:"policy,omitempty"`
	Location       *Reference                  `bson:"location,omitempty" json:"location,omitempty"`
	Reason         []Coding                    `bson:"reason,omitempty" json:"reason,omitempty"`
	Activity       *Coding                     `bson:"activity,omitempty" json:"activity,omitempty"`
	Agent          []ProvenanceAgentComponent  `bson:"agent,omitempty" json:"agent,omitempty"`
	Entity         []ProvenanceEntityComponent `bson:"entity,omitempty" json:"entity,omitempty"`
	Signature      []Signature                 `bson:"signature,omitempty" json:"signature,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Provenance) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Provenance"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Provenance), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Provenance) GetBSON() (interface{}, error) {
	x.ResourceType = "Provenance"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "provenance" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type provenance Provenance

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Provenance) UnmarshalJSON(data []byte) (err error) {
	x2 := provenance{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Provenance(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Provenance) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Provenance"
	} else if x.ResourceType != "Provenance" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Provenance, instead received %s", x.ResourceType))
	}
	return nil
}

type ProvenanceAgentComponent struct {
	BackboneElement     `bson:",inline"`
	Role                []CodeableConcept `bson:"role,omitempty" json:"role,omitempty"`
	WhoUri              string            `bson:"whoUri,omitempty" json:"whoUri,omitempty"`
	WhoReference        *Reference        `bson:"whoReference,omitempty" json:"whoReference,omitempty"`
	OnBehalfOfUri       string            `bson:"onBehalfOfUri,omitempty" json:"onBehalfOfUri,omitempty"`
	OnBehalfOfReference *Reference        `bson:"onBehalfOfReference,omitempty" json:"onBehalfOfReference,omitempty"`
	RelatedAgentType    *CodeableConcept  `bson:"relatedAgentType,omitempty" json:"relatedAgentType,omitempty"`
}

type ProvenanceEntityComponent struct {
	BackboneElement `bson:",inline"`
	Role            string                     `bson:"role,omitempty" json:"role,omitempty"`
	WhatUri         string                     `bson:"whatUri,omitempty" json:"whatUri,omitempty"`
	WhatReference   *Reference                 `bson:"whatReference,omitempty" json:"whatReference,omitempty"`
	WhatIdentifier  *Identifier                `bson:"whatIdentifier,omitempty" json:"whatIdentifier,omitempty"`
	Agent           []ProvenanceAgentComponent `bson:"agent,omitempty" json:"agent,omitempty"`
}
