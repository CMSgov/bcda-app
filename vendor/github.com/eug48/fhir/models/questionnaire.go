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

type Questionnaire struct {
	DomainResource  `bson:",inline"`
	Url             string                       `bson:"url,omitempty" json:"url,omitempty"`
	Identifier      []Identifier                 `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Version         string                       `bson:"version,omitempty" json:"version,omitempty"`
	Name            string                       `bson:"name,omitempty" json:"name,omitempty"`
	Title           string                       `bson:"title,omitempty" json:"title,omitempty"`
	Status          string                       `bson:"status,omitempty" json:"status,omitempty"`
	Experimental    *bool                        `bson:"experimental,omitempty" json:"experimental,omitempty"`
	Date            *FHIRDateTime                `bson:"date,omitempty" json:"date,omitempty"`
	Publisher       string                       `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Description     string                       `bson:"description,omitempty" json:"description,omitempty"`
	Purpose         string                       `bson:"purpose,omitempty" json:"purpose,omitempty"`
	ApprovalDate    *FHIRDateTime                `bson:"approvalDate,omitempty" json:"approvalDate,omitempty"`
	LastReviewDate  *FHIRDateTime                `bson:"lastReviewDate,omitempty" json:"lastReviewDate,omitempty"`
	EffectivePeriod *Period                      `bson:"effectivePeriod,omitempty" json:"effectivePeriod,omitempty"`
	UseContext      []UsageContext               `bson:"useContext,omitempty" json:"useContext,omitempty"`
	Jurisdiction    []CodeableConcept            `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	Contact         []ContactDetail              `bson:"contact,omitempty" json:"contact,omitempty"`
	Copyright       string                       `bson:"copyright,omitempty" json:"copyright,omitempty"`
	Code            []Coding                     `bson:"code,omitempty" json:"code,omitempty"`
	SubjectType     []string                     `bson:"subjectType,omitempty" json:"subjectType,omitempty"`
	Item            []QuestionnaireItemComponent `bson:"item,omitempty" json:"item,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Questionnaire) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Questionnaire"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Questionnaire), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Questionnaire) GetBSON() (interface{}, error) {
	x.ResourceType = "Questionnaire"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "questionnaire" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type questionnaire Questionnaire

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Questionnaire) UnmarshalJSON(data []byte) (err error) {
	x2 := questionnaire{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Questionnaire(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Questionnaire) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Questionnaire"
	} else if x.ResourceType != "Questionnaire" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Questionnaire, instead received %s", x.ResourceType))
	}
	return nil
}

type QuestionnaireItemComponent struct {
	BackboneElement   `bson:",inline"`
	LinkId            string                                 `bson:"linkId,omitempty" json:"linkId,omitempty"`
	Definition        string                                 `bson:"definition,omitempty" json:"definition,omitempty"`
	Code              []Coding                               `bson:"code,omitempty" json:"code,omitempty"`
	Prefix            string                                 `bson:"prefix,omitempty" json:"prefix,omitempty"`
	Text              string                                 `bson:"text,omitempty" json:"text,omitempty"`
	Type              string                                 `bson:"type,omitempty" json:"type,omitempty"`
	EnableWhen        []QuestionnaireItemEnableWhenComponent `bson:"enableWhen,omitempty" json:"enableWhen,omitempty"`
	Required          *bool                                  `bson:"required,omitempty" json:"required,omitempty"`
	Repeats           *bool                                  `bson:"repeats,omitempty" json:"repeats,omitempty"`
	ReadOnly          *bool                                  `bson:"readOnly,omitempty" json:"readOnly,omitempty"`
	MaxLength         *int32                                 `bson:"maxLength,omitempty" json:"maxLength,omitempty"`
	Options           *Reference                             `bson:"options,omitempty" json:"options,omitempty"`
	Option            []QuestionnaireItemOptionComponent     `bson:"option,omitempty" json:"option,omitempty"`
	InitialBoolean    *bool                                  `bson:"initialBoolean,omitempty" json:"initialBoolean,omitempty"`
	InitialDecimal    *float64                               `bson:"initialDecimal,omitempty" json:"initialDecimal,omitempty"`
	InitialInteger    *int32                                 `bson:"initialInteger,omitempty" json:"initialInteger,omitempty"`
	InitialDate       *FHIRDateTime                          `bson:"initialDate,omitempty" json:"initialDate,omitempty"`
	InitialDateTime   *FHIRDateTime                          `bson:"initialDateTime,omitempty" json:"initialDateTime,omitempty"`
	InitialTime       *FHIRDateTime                          `bson:"initialTime,omitempty" json:"initialTime,omitempty"`
	InitialString     string                                 `bson:"initialString,omitempty" json:"initialString,omitempty"`
	InitialUri        string                                 `bson:"initialUri,omitempty" json:"initialUri,omitempty"`
	InitialAttachment *Attachment                            `bson:"initialAttachment,omitempty" json:"initialAttachment,omitempty"`
	InitialCoding     *Coding                                `bson:"initialCoding,omitempty" json:"initialCoding,omitempty"`
	InitialQuantity   *Quantity                              `bson:"initialQuantity,omitempty" json:"initialQuantity,omitempty"`
	InitialReference  *Reference                             `bson:"initialReference,omitempty" json:"initialReference,omitempty"`
	Item              []QuestionnaireItemComponent           `bson:"item,omitempty" json:"item,omitempty"`
}

type QuestionnaireItemEnableWhenComponent struct {
	BackboneElement  `bson:",inline"`
	Question         string        `bson:"question,omitempty" json:"question,omitempty"`
	HasAnswer        *bool         `bson:"hasAnswer,omitempty" json:"hasAnswer,omitempty"`
	AnswerBoolean    *bool         `bson:"answerBoolean,omitempty" json:"answerBoolean,omitempty"`
	AnswerDecimal    *float64      `bson:"answerDecimal,omitempty" json:"answerDecimal,omitempty"`
	AnswerInteger    *int32        `bson:"answerInteger,omitempty" json:"answerInteger,omitempty"`
	AnswerDate       *FHIRDateTime `bson:"answerDate,omitempty" json:"answerDate,omitempty"`
	AnswerDateTime   *FHIRDateTime `bson:"answerDateTime,omitempty" json:"answerDateTime,omitempty"`
	AnswerTime       *FHIRDateTime `bson:"answerTime,omitempty" json:"answerTime,omitempty"`
	AnswerString     string        `bson:"answerString,omitempty" json:"answerString,omitempty"`
	AnswerUri        string        `bson:"answerUri,omitempty" json:"answerUri,omitempty"`
	AnswerAttachment *Attachment   `bson:"answerAttachment,omitempty" json:"answerAttachment,omitempty"`
	AnswerCoding     *Coding       `bson:"answerCoding,omitempty" json:"answerCoding,omitempty"`
	AnswerQuantity   *Quantity     `bson:"answerQuantity,omitempty" json:"answerQuantity,omitempty"`
	AnswerReference  *Reference    `bson:"answerReference,omitempty" json:"answerReference,omitempty"`
}

type QuestionnaireItemOptionComponent struct {
	BackboneElement `bson:",inline"`
	ValueInteger    *int32        `bson:"valueInteger,omitempty" json:"valueInteger,omitempty"`
	ValueDate       *FHIRDateTime `bson:"valueDate,omitempty" json:"valueDate,omitempty"`
	ValueTime       *FHIRDateTime `bson:"valueTime,omitempty" json:"valueTime,omitempty"`
	ValueString     string        `bson:"valueString,omitempty" json:"valueString,omitempty"`
	ValueCoding     *Coding       `bson:"valueCoding,omitempty" json:"valueCoding,omitempty"`
}
