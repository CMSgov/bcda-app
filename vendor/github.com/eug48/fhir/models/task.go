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

type Task struct {
	DomainResource      `bson:",inline"`
	Identifier          []Identifier              `bson:"identifier,omitempty" json:"identifier,omitempty"`
	DefinitionUri       string                    `bson:"definitionUri,omitempty" json:"definitionUri,omitempty"`
	DefinitionReference *Reference                `bson:"definitionReference,omitempty" json:"definitionReference,omitempty"`
	BasedOn             []Reference               `bson:"basedOn,omitempty" json:"basedOn,omitempty"`
	GroupIdentifier     *Identifier               `bson:"groupIdentifier,omitempty" json:"groupIdentifier,omitempty"`
	PartOf              []Reference               `bson:"partOf,omitempty" json:"partOf,omitempty"`
	Status              string                    `bson:"status,omitempty" json:"status,omitempty"`
	StatusReason        *CodeableConcept          `bson:"statusReason,omitempty" json:"statusReason,omitempty"`
	BusinessStatus      *CodeableConcept          `bson:"businessStatus,omitempty" json:"businessStatus,omitempty"`
	Intent              string                    `bson:"intent,omitempty" json:"intent,omitempty"`
	Priority            string                    `bson:"priority,omitempty" json:"priority,omitempty"`
	Code                *CodeableConcept          `bson:"code,omitempty" json:"code,omitempty"`
	Description         string                    `bson:"description,omitempty" json:"description,omitempty"`
	Focus               *Reference                `bson:"focus,omitempty" json:"focus,omitempty"`
	For                 *Reference                `bson:"for,omitempty" json:"for,omitempty"`
	Context             *Reference                `bson:"context,omitempty" json:"context,omitempty"`
	ExecutionPeriod     *Period                   `bson:"executionPeriod,omitempty" json:"executionPeriod,omitempty"`
	AuthoredOn          *FHIRDateTime             `bson:"authoredOn,omitempty" json:"authoredOn,omitempty"`
	LastModified        *FHIRDateTime             `bson:"lastModified,omitempty" json:"lastModified,omitempty"`
	Requester           *TaskRequesterComponent   `bson:"requester,omitempty" json:"requester,omitempty"`
	PerformerType       []CodeableConcept         `bson:"performerType,omitempty" json:"performerType,omitempty"`
	Owner               *Reference                `bson:"owner,omitempty" json:"owner,omitempty"`
	Reason              *CodeableConcept          `bson:"reason,omitempty" json:"reason,omitempty"`
	Note                []Annotation              `bson:"note,omitempty" json:"note,omitempty"`
	RelevantHistory     []Reference               `bson:"relevantHistory,omitempty" json:"relevantHistory,omitempty"`
	Restriction         *TaskRestrictionComponent `bson:"restriction,omitempty" json:"restriction,omitempty"`
	Input               []TaskParameterComponent  `bson:"input,omitempty" json:"input,omitempty"`
	Output              []TaskOutputComponent     `bson:"output,omitempty" json:"output,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Task) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Task"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Task), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Task) GetBSON() (interface{}, error) {
	x.ResourceType = "Task"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "task" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type task Task

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Task) UnmarshalJSON(data []byte) (err error) {
	x2 := task{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Task(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Task) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Task"
	} else if x.ResourceType != "Task" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Task, instead received %s", x.ResourceType))
	}
	return nil
}

type TaskRequesterComponent struct {
	BackboneElement `bson:",inline"`
	Agent           *Reference `bson:"agent,omitempty" json:"agent,omitempty"`
	OnBehalfOf      *Reference `bson:"onBehalfOf,omitempty" json:"onBehalfOf,omitempty"`
}

type TaskRestrictionComponent struct {
	BackboneElement `bson:",inline"`
	Repetitions     *uint32     `bson:"repetitions,omitempty" json:"repetitions,omitempty"`
	Period          *Period     `bson:"period,omitempty" json:"period,omitempty"`
	Recipient       []Reference `bson:"recipient,omitempty" json:"recipient,omitempty"`
}

type TaskParameterComponent struct {
	BackboneElement      `bson:",inline"`
	Type                 *CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	ValueAddress         *Address         `bson:"valueAddress,omitempty" json:"valueAddress,omitempty"`
	ValueAnnotation      *Annotation      `bson:"valueAnnotation,omitempty" json:"valueAnnotation,omitempty"`
	ValueAttachment      *Attachment      `bson:"valueAttachment,omitempty" json:"valueAttachment,omitempty"`
	ValueBase64Binary    string           `bson:"valueBase64Binary,omitempty" json:"valueBase64Binary,omitempty"`
	ValueBoolean         *bool            `bson:"valueBoolean,omitempty" json:"valueBoolean,omitempty"`
	ValueCode            string           `bson:"valueCode,omitempty" json:"valueCode,omitempty"`
	ValueCodeableConcept *CodeableConcept `bson:"valueCodeableConcept,omitempty" json:"valueCodeableConcept,omitempty"`
	ValueCoding          *Coding          `bson:"valueCoding,omitempty" json:"valueCoding,omitempty"`
	ValueContactPoint    *ContactPoint    `bson:"valueContactPoint,omitempty" json:"valueContactPoint,omitempty"`
	ValueDate            *FHIRDateTime    `bson:"valueDate,omitempty" json:"valueDate,omitempty"`
	ValueDateTime        *FHIRDateTime    `bson:"valueDateTime,omitempty" json:"valueDateTime,omitempty"`
	ValueDecimal         *float64         `bson:"valueDecimal,omitempty" json:"valueDecimal,omitempty"`
	ValueHumanName       *HumanName       `bson:"valueHumanName,omitempty" json:"valueHumanName,omitempty"`
	ValueId              string           `bson:"valueId,omitempty" json:"valueId,omitempty"`
	ValueIdentifier      *Identifier      `bson:"valueIdentifier,omitempty" json:"valueIdentifier,omitempty"`
	ValueInstant         *FHIRDateTime    `bson:"valueInstant,omitempty" json:"valueInstant,omitempty"`
	ValueInteger         *int32           `bson:"valueInteger,omitempty" json:"valueInteger,omitempty"`
	ValueMarkdown        string           `bson:"valueMarkdown,omitempty" json:"valueMarkdown,omitempty"`
	ValueMeta            *Meta            `bson:"valueMeta,omitempty" json:"valueMeta,omitempty"`
	ValueOid             string           `bson:"valueOid,omitempty" json:"valueOid,omitempty"`
	ValuePeriod          *Period          `bson:"valuePeriod,omitempty" json:"valuePeriod,omitempty"`
	ValuePositiveInt     *uint32          `bson:"valuePositiveInt,omitempty" json:"valuePositiveInt,omitempty"`
	ValueQuantity        *Quantity        `bson:"valueQuantity,omitempty" json:"valueQuantity,omitempty"`
	ValueRange           *Range           `bson:"valueRange,omitempty" json:"valueRange,omitempty"`
	ValueRatio           *Ratio           `bson:"valueRatio,omitempty" json:"valueRatio,omitempty"`
	ValueReference       *Reference       `bson:"valueReference,omitempty" json:"valueReference,omitempty"`
	ValueSampledData     *SampledData     `bson:"valueSampledData,omitempty" json:"valueSampledData,omitempty"`
	ValueSignature       *Signature       `bson:"valueSignature,omitempty" json:"valueSignature,omitempty"`
	ValueString          string           `bson:"valueString,omitempty" json:"valueString,omitempty"`
	ValueTime            *FHIRDateTime    `bson:"valueTime,omitempty" json:"valueTime,omitempty"`
	ValueTiming          *Timing          `bson:"valueTiming,omitempty" json:"valueTiming,omitempty"`
	ValueUnsignedInt     *uint32          `bson:"valueUnsignedInt,omitempty" json:"valueUnsignedInt,omitempty"`
	ValueUri             string           `bson:"valueUri,omitempty" json:"valueUri,omitempty"`
}

type TaskOutputComponent struct {
	BackboneElement      `bson:",inline"`
	Type                 *CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	ValueAddress         *Address         `bson:"valueAddress,omitempty" json:"valueAddress,omitempty"`
	ValueAnnotation      *Annotation      `bson:"valueAnnotation,omitempty" json:"valueAnnotation,omitempty"`
	ValueAttachment      *Attachment      `bson:"valueAttachment,omitempty" json:"valueAttachment,omitempty"`
	ValueBase64Binary    string           `bson:"valueBase64Binary,omitempty" json:"valueBase64Binary,omitempty"`
	ValueBoolean         *bool            `bson:"valueBoolean,omitempty" json:"valueBoolean,omitempty"`
	ValueCode            string           `bson:"valueCode,omitempty" json:"valueCode,omitempty"`
	ValueCodeableConcept *CodeableConcept `bson:"valueCodeableConcept,omitempty" json:"valueCodeableConcept,omitempty"`
	ValueCoding          *Coding          `bson:"valueCoding,omitempty" json:"valueCoding,omitempty"`
	ValueContactPoint    *ContactPoint    `bson:"valueContactPoint,omitempty" json:"valueContactPoint,omitempty"`
	ValueDate            *FHIRDateTime    `bson:"valueDate,omitempty" json:"valueDate,omitempty"`
	ValueDateTime        *FHIRDateTime    `bson:"valueDateTime,omitempty" json:"valueDateTime,omitempty"`
	ValueDecimal         *float64         `bson:"valueDecimal,omitempty" json:"valueDecimal,omitempty"`
	ValueHumanName       *HumanName       `bson:"valueHumanName,omitempty" json:"valueHumanName,omitempty"`
	ValueId              string           `bson:"valueId,omitempty" json:"valueId,omitempty"`
	ValueIdentifier      *Identifier      `bson:"valueIdentifier,omitempty" json:"valueIdentifier,omitempty"`
	ValueInstant         *FHIRDateTime    `bson:"valueInstant,omitempty" json:"valueInstant,omitempty"`
	ValueInteger         *int32           `bson:"valueInteger,omitempty" json:"valueInteger,omitempty"`
	ValueMarkdown        string           `bson:"valueMarkdown,omitempty" json:"valueMarkdown,omitempty"`
	ValueMeta            *Meta            `bson:"valueMeta,omitempty" json:"valueMeta,omitempty"`
	ValueOid             string           `bson:"valueOid,omitempty" json:"valueOid,omitempty"`
	ValuePeriod          *Period          `bson:"valuePeriod,omitempty" json:"valuePeriod,omitempty"`
	ValuePositiveInt     *uint32          `bson:"valuePositiveInt,omitempty" json:"valuePositiveInt,omitempty"`
	ValueQuantity        *Quantity        `bson:"valueQuantity,omitempty" json:"valueQuantity,omitempty"`
	ValueRange           *Range           `bson:"valueRange,omitempty" json:"valueRange,omitempty"`
	ValueRatio           *Ratio           `bson:"valueRatio,omitempty" json:"valueRatio,omitempty"`
	ValueReference       *Reference       `bson:"valueReference,omitempty" json:"valueReference,omitempty"`
	ValueSampledData     *SampledData     `bson:"valueSampledData,omitempty" json:"valueSampledData,omitempty"`
	ValueSignature       *Signature       `bson:"valueSignature,omitempty" json:"valueSignature,omitempty"`
	ValueString          string           `bson:"valueString,omitempty" json:"valueString,omitempty"`
	ValueTime            *FHIRDateTime    `bson:"valueTime,omitempty" json:"valueTime,omitempty"`
	ValueTiming          *Timing          `bson:"valueTiming,omitempty" json:"valueTiming,omitempty"`
	ValueUnsignedInt     *uint32          `bson:"valueUnsignedInt,omitempty" json:"valueUnsignedInt,omitempty"`
	ValueUri             string           `bson:"valueUri,omitempty" json:"valueUri,omitempty"`
}
