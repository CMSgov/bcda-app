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

type Consent struct {
	DomainResource   `bson:",inline"`
	Identifier       *Identifier              `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status           string                   `bson:"status,omitempty" json:"status,omitempty"`
	Category         []CodeableConcept        `bson:"category,omitempty" json:"category,omitempty"`
	Patient          *Reference               `bson:"patient,omitempty" json:"patient,omitempty"`
	Period           *Period                  `bson:"period,omitempty" json:"period,omitempty"`
	DateTime         *FHIRDateTime            `bson:"dateTime,omitempty" json:"dateTime,omitempty"`
	ConsentingParty  []Reference              `bson:"consentingParty,omitempty" json:"consentingParty,omitempty"`
	Actor            []ConsentActorComponent  `bson:"actor,omitempty" json:"actor,omitempty"`
	Action           []CodeableConcept        `bson:"action,omitempty" json:"action,omitempty"`
	Organization     []Reference              `bson:"organization,omitempty" json:"organization,omitempty"`
	SourceAttachment *Attachment              `bson:"sourceAttachment,omitempty" json:"sourceAttachment,omitempty"`
	SourceIdentifier *Identifier              `bson:"sourceIdentifier,omitempty" json:"sourceIdentifier,omitempty"`
	SourceReference  *Reference               `bson:"sourceReference,omitempty" json:"sourceReference,omitempty"`
	Policy           []ConsentPolicyComponent `bson:"policy,omitempty" json:"policy,omitempty"`
	PolicyRule       string                   `bson:"policyRule,omitempty" json:"policyRule,omitempty"`
	SecurityLabel    []Coding                 `bson:"securityLabel,omitempty" json:"securityLabel,omitempty"`
	Purpose          []Coding                 `bson:"purpose,omitempty" json:"purpose,omitempty"`
	DataPeriod       *Period                  `bson:"dataPeriod,omitempty" json:"dataPeriod,omitempty"`
	Data             []ConsentDataComponent   `bson:"data,omitempty" json:"data,omitempty"`
	Except           []ConsentExceptComponent `bson:"except,omitempty" json:"except,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Consent) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Consent"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Consent), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Consent) GetBSON() (interface{}, error) {
	x.ResourceType = "Consent"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "consent" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type consent Consent

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Consent) UnmarshalJSON(data []byte) (err error) {
	x2 := consent{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Consent(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Consent) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Consent"
	} else if x.ResourceType != "Consent" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Consent, instead received %s", x.ResourceType))
	}
	return nil
}

type ConsentActorComponent struct {
	BackboneElement `bson:",inline"`
	Role            *CodeableConcept `bson:"role,omitempty" json:"role,omitempty"`
	Reference       *Reference       `bson:"reference,omitempty" json:"reference,omitempty"`
}

type ConsentPolicyComponent struct {
	BackboneElement `bson:",inline"`
	Authority       string `bson:"authority,omitempty" json:"authority,omitempty"`
	Uri             string `bson:"uri,omitempty" json:"uri,omitempty"`
}

type ConsentDataComponent struct {
	BackboneElement `bson:",inline"`
	Meaning         string     `bson:"meaning,omitempty" json:"meaning,omitempty"`
	Reference       *Reference `bson:"reference,omitempty" json:"reference,omitempty"`
}

type ConsentExceptComponent struct {
	BackboneElement `bson:",inline"`
	Type            string                        `bson:"type,omitempty" json:"type,omitempty"`
	Period          *Period                       `bson:"period,omitempty" json:"period,omitempty"`
	Actor           []ConsentExceptActorComponent `bson:"actor,omitempty" json:"actor,omitempty"`
	Action          []CodeableConcept             `bson:"action,omitempty" json:"action,omitempty"`
	SecurityLabel   []Coding                      `bson:"securityLabel,omitempty" json:"securityLabel,omitempty"`
	Purpose         []Coding                      `bson:"purpose,omitempty" json:"purpose,omitempty"`
	Class           []Coding                      `bson:"class,omitempty" json:"class,omitempty"`
	Code            []Coding                      `bson:"code,omitempty" json:"code,omitempty"`
	DataPeriod      *Period                       `bson:"dataPeriod,omitempty" json:"dataPeriod,omitempty"`
	Data            []ConsentExceptDataComponent  `bson:"data,omitempty" json:"data,omitempty"`
}

type ConsentExceptActorComponent struct {
	BackboneElement `bson:",inline"`
	Role            *CodeableConcept `bson:"role,omitempty" json:"role,omitempty"`
	Reference       *Reference       `bson:"reference,omitempty" json:"reference,omitempty"`
}

type ConsentExceptDataComponent struct {
	BackboneElement `bson:",inline"`
	Meaning         string     `bson:"meaning,omitempty" json:"meaning,omitempty"`
	Reference       *Reference `bson:"reference,omitempty" json:"reference,omitempty"`
}
