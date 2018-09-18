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

type Contract struct {
	DomainResource    `bson:",inline"`
	Identifier        *Identifier                           `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status            string                                `bson:"status,omitempty" json:"status,omitempty"`
	Issued            *FHIRDateTime                         `bson:"issued,omitempty" json:"issued,omitempty"`
	Applies           *Period                               `bson:"applies,omitempty" json:"applies,omitempty"`
	Subject           []Reference                           `bson:"subject,omitempty" json:"subject,omitempty"`
	Topic             []Reference                           `bson:"topic,omitempty" json:"topic,omitempty"`
	Authority         []Reference                           `bson:"authority,omitempty" json:"authority,omitempty"`
	Domain            []Reference                           `bson:"domain,omitempty" json:"domain,omitempty"`
	Type              *CodeableConcept                      `bson:"type,omitempty" json:"type,omitempty"`
	SubType           []CodeableConcept                     `bson:"subType,omitempty" json:"subType,omitempty"`
	Action            []CodeableConcept                     `bson:"action,omitempty" json:"action,omitempty"`
	ActionReason      []CodeableConcept                     `bson:"actionReason,omitempty" json:"actionReason,omitempty"`
	DecisionType      *CodeableConcept                      `bson:"decisionType,omitempty" json:"decisionType,omitempty"`
	ContentDerivative *CodeableConcept                      `bson:"contentDerivative,omitempty" json:"contentDerivative,omitempty"`
	SecurityLabel     []Coding                              `bson:"securityLabel,omitempty" json:"securityLabel,omitempty"`
	Agent             []ContractAgentComponent              `bson:"agent,omitempty" json:"agent,omitempty"`
	Signer            []ContractSignatoryComponent          `bson:"signer,omitempty" json:"signer,omitempty"`
	ValuedItem        []ContractValuedItemComponent         `bson:"valuedItem,omitempty" json:"valuedItem,omitempty"`
	Term              []ContractTermComponent               `bson:"term,omitempty" json:"term,omitempty"`
	BindingAttachment *Attachment                           `bson:"bindingAttachment,omitempty" json:"bindingAttachment,omitempty"`
	BindingReference  *Reference                            `bson:"bindingReference,omitempty" json:"bindingReference,omitempty"`
	Friendly          []ContractFriendlyLanguageComponent   `bson:"friendly,omitempty" json:"friendly,omitempty"`
	Legal             []ContractLegalLanguageComponent      `bson:"legal,omitempty" json:"legal,omitempty"`
	Rule              []ContractComputableLanguageComponent `bson:"rule,omitempty" json:"rule,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Contract) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Contract"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Contract), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Contract) GetBSON() (interface{}, error) {
	x.ResourceType = "Contract"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "contract" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type contract Contract

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Contract) UnmarshalJSON(data []byte) (err error) {
	x2 := contract{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Contract(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Contract) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Contract"
	} else if x.ResourceType != "Contract" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Contract, instead received %s", x.ResourceType))
	}
	return nil
}

type ContractAgentComponent struct {
	BackboneElement `bson:",inline"`
	Actor           *Reference        `bson:"actor,omitempty" json:"actor,omitempty"`
	Role            []CodeableConcept `bson:"role,omitempty" json:"role,omitempty"`
}

type ContractSignatoryComponent struct {
	BackboneElement `bson:",inline"`
	Type            *Coding     `bson:"type,omitempty" json:"type,omitempty"`
	Party           *Reference  `bson:"party,omitempty" json:"party,omitempty"`
	Signature       []Signature `bson:"signature,omitempty" json:"signature,omitempty"`
}

type ContractValuedItemComponent struct {
	BackboneElement       `bson:",inline"`
	EntityCodeableConcept *CodeableConcept `bson:"entityCodeableConcept,omitempty" json:"entityCodeableConcept,omitempty"`
	EntityReference       *Reference       `bson:"entityReference,omitempty" json:"entityReference,omitempty"`
	Identifier            *Identifier      `bson:"identifier,omitempty" json:"identifier,omitempty"`
	EffectiveTime         *FHIRDateTime    `bson:"effectiveTime,omitempty" json:"effectiveTime,omitempty"`
	Quantity              *Quantity        `bson:"quantity,omitempty" json:"quantity,omitempty"`
	UnitPrice             *Quantity        `bson:"unitPrice,omitempty" json:"unitPrice,omitempty"`
	Factor                *float64         `bson:"factor,omitempty" json:"factor,omitempty"`
	Points                *float64         `bson:"points,omitempty" json:"points,omitempty"`
	Net                   *Quantity        `bson:"net,omitempty" json:"net,omitempty"`
}

type ContractTermComponent struct {
	BackboneElement `bson:",inline"`
	Identifier      *Identifier                       `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Issued          *FHIRDateTime                     `bson:"issued,omitempty" json:"issued,omitempty"`
	Applies         *Period                           `bson:"applies,omitempty" json:"applies,omitempty"`
	Type            *CodeableConcept                  `bson:"type,omitempty" json:"type,omitempty"`
	SubType         *CodeableConcept                  `bson:"subType,omitempty" json:"subType,omitempty"`
	Topic           []Reference                       `bson:"topic,omitempty" json:"topic,omitempty"`
	Action          []CodeableConcept                 `bson:"action,omitempty" json:"action,omitempty"`
	ActionReason    []CodeableConcept                 `bson:"actionReason,omitempty" json:"actionReason,omitempty"`
	SecurityLabel   []Coding                          `bson:"securityLabel,omitempty" json:"securityLabel,omitempty"`
	Agent           []ContractTermAgentComponent      `bson:"agent,omitempty" json:"agent,omitempty"`
	Text            string                            `bson:"text,omitempty" json:"text,omitempty"`
	ValuedItem      []ContractTermValuedItemComponent `bson:"valuedItem,omitempty" json:"valuedItem,omitempty"`
	Group           []ContractTermComponent           `bson:"group,omitempty" json:"group,omitempty"`
}

type ContractTermAgentComponent struct {
	BackboneElement `bson:",inline"`
	Actor           *Reference        `bson:"actor,omitempty" json:"actor,omitempty"`
	Role            []CodeableConcept `bson:"role,omitempty" json:"role,omitempty"`
}

type ContractTermValuedItemComponent struct {
	BackboneElement       `bson:",inline"`
	EntityCodeableConcept *CodeableConcept `bson:"entityCodeableConcept,omitempty" json:"entityCodeableConcept,omitempty"`
	EntityReference       *Reference       `bson:"entityReference,omitempty" json:"entityReference,omitempty"`
	Identifier            *Identifier      `bson:"identifier,omitempty" json:"identifier,omitempty"`
	EffectiveTime         *FHIRDateTime    `bson:"effectiveTime,omitempty" json:"effectiveTime,omitempty"`
	Quantity              *Quantity        `bson:"quantity,omitempty" json:"quantity,omitempty"`
	UnitPrice             *Quantity        `bson:"unitPrice,omitempty" json:"unitPrice,omitempty"`
	Factor                *float64         `bson:"factor,omitempty" json:"factor,omitempty"`
	Points                *float64         `bson:"points,omitempty" json:"points,omitempty"`
	Net                   *Quantity        `bson:"net,omitempty" json:"net,omitempty"`
}

type ContractFriendlyLanguageComponent struct {
	BackboneElement   `bson:",inline"`
	ContentAttachment *Attachment `bson:"contentAttachment,omitempty" json:"contentAttachment,omitempty"`
	ContentReference  *Reference  `bson:"contentReference,omitempty" json:"contentReference,omitempty"`
}

type ContractLegalLanguageComponent struct {
	BackboneElement   `bson:",inline"`
	ContentAttachment *Attachment `bson:"contentAttachment,omitempty" json:"contentAttachment,omitempty"`
	ContentReference  *Reference  `bson:"contentReference,omitempty" json:"contentReference,omitempty"`
}

type ContractComputableLanguageComponent struct {
	BackboneElement   `bson:",inline"`
	ContentAttachment *Attachment `bson:"contentAttachment,omitempty" json:"contentAttachment,omitempty"`
	ContentReference  *Reference  `bson:"contentReference,omitempty" json:"contentReference,omitempty"`
}
