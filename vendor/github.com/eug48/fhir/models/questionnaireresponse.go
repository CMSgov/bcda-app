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

type QuestionnaireResponse struct {
	DomainResource `bson:",inline"`
	Identifier     *Identifier                          `bson:"identifier,omitempty" json:"identifier,omitempty"`
	BasedOn        []Reference                          `bson:"basedOn,omitempty" json:"basedOn,omitempty"`
	Parent         []Reference                          `bson:"parent,omitempty" json:"parent,omitempty"`
	Questionnaire  *Reference                           `bson:"questionnaire,omitempty" json:"questionnaire,omitempty"`
	Status         string                               `bson:"status,omitempty" json:"status,omitempty"`
	Subject        *Reference                           `bson:"subject,omitempty" json:"subject,omitempty"`
	Context        *Reference                           `bson:"context,omitempty" json:"context,omitempty"`
	Authored       *FHIRDateTime                        `bson:"authored,omitempty" json:"authored,omitempty"`
	Author         *Reference                           `bson:"author,omitempty" json:"author,omitempty"`
	Source         *Reference                           `bson:"source,omitempty" json:"source,omitempty"`
	Item           []QuestionnaireResponseItemComponent `bson:"item,omitempty" json:"item,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *QuestionnaireResponse) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "QuestionnaireResponse"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to QuestionnaireResponse), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *QuestionnaireResponse) GetBSON() (interface{}, error) {
	x.ResourceType = "QuestionnaireResponse"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "questionnaireResponse" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type questionnaireResponse QuestionnaireResponse

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *QuestionnaireResponse) UnmarshalJSON(data []byte) (err error) {
	x2 := questionnaireResponse{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = QuestionnaireResponse(x2)
		return x.checkResourceType()
	}
	return
}

func (x *QuestionnaireResponse) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "QuestionnaireResponse"
	} else if x.ResourceType != "QuestionnaireResponse" {
		return errors.New(fmt.Sprintf("Expected resourceType to be QuestionnaireResponse, instead received %s", x.ResourceType))
	}
	return nil
}

type QuestionnaireResponseItemComponent struct {
	BackboneElement `bson:",inline"`
	LinkId          string                                     `bson:"linkId,omitempty" json:"linkId,omitempty"`
	Definition      string                                     `bson:"definition,omitempty" json:"definition,omitempty"`
	Text            string                                     `bson:"text,omitempty" json:"text,omitempty"`
	Subject         *Reference                                 `bson:"subject,omitempty" json:"subject,omitempty"`
	Answer          []QuestionnaireResponseItemAnswerComponent `bson:"answer,omitempty" json:"answer,omitempty"`
	Item            []QuestionnaireResponseItemComponent       `bson:"item,omitempty" json:"item,omitempty"`
}

type QuestionnaireResponseItemAnswerComponent struct {
	BackboneElement `bson:",inline"`
	ValueBoolean    *bool                                `bson:"valueBoolean,omitempty" json:"valueBoolean,omitempty"`
	ValueDecimal    *float64                             `bson:"valueDecimal,omitempty" json:"valueDecimal,omitempty"`
	ValueInteger    *int32                               `bson:"valueInteger,omitempty" json:"valueInteger,omitempty"`
	ValueDate       *FHIRDateTime                        `bson:"valueDate,omitempty" json:"valueDate,omitempty"`
	ValueDateTime   *FHIRDateTime                        `bson:"valueDateTime,omitempty" json:"valueDateTime,omitempty"`
	ValueTime       *FHIRDateTime                        `bson:"valueTime,omitempty" json:"valueTime,omitempty"`
	ValueString     string                               `bson:"valueString,omitempty" json:"valueString,omitempty"`
	ValueUri        string                               `bson:"valueUri,omitempty" json:"valueUri,omitempty"`
	ValueAttachment *Attachment                          `bson:"valueAttachment,omitempty" json:"valueAttachment,omitempty"`
	ValueCoding     *Coding                              `bson:"valueCoding,omitempty" json:"valueCoding,omitempty"`
	ValueQuantity   *Quantity                            `bson:"valueQuantity,omitempty" json:"valueQuantity,omitempty"`
	ValueReference  *Reference                           `bson:"valueReference,omitempty" json:"valueReference,omitempty"`
	Item            []QuestionnaireResponseItemComponent `bson:"item,omitempty" json:"item,omitempty"`
}
