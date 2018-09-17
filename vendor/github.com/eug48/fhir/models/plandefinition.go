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

type PlanDefinition struct {
	DomainResource  `bson:",inline"`
	Url             string                          `bson:"url,omitempty" json:"url,omitempty"`
	Identifier      []Identifier                    `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Version         string                          `bson:"version,omitempty" json:"version,omitempty"`
	Name            string                          `bson:"name,omitempty" json:"name,omitempty"`
	Title           string                          `bson:"title,omitempty" json:"title,omitempty"`
	Type            *CodeableConcept                `bson:"type,omitempty" json:"type,omitempty"`
	Status          string                          `bson:"status,omitempty" json:"status,omitempty"`
	Experimental    *bool                           `bson:"experimental,omitempty" json:"experimental,omitempty"`
	Date            *FHIRDateTime                   `bson:"date,omitempty" json:"date,omitempty"`
	Publisher       string                          `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Description     string                          `bson:"description,omitempty" json:"description,omitempty"`
	Purpose         string                          `bson:"purpose,omitempty" json:"purpose,omitempty"`
	Usage           string                          `bson:"usage,omitempty" json:"usage,omitempty"`
	ApprovalDate    *FHIRDateTime                   `bson:"approvalDate,omitempty" json:"approvalDate,omitempty"`
	LastReviewDate  *FHIRDateTime                   `bson:"lastReviewDate,omitempty" json:"lastReviewDate,omitempty"`
	EffectivePeriod *Period                         `bson:"effectivePeriod,omitempty" json:"effectivePeriod,omitempty"`
	UseContext      []UsageContext                  `bson:"useContext,omitempty" json:"useContext,omitempty"`
	Jurisdiction    []CodeableConcept               `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	Topic           []CodeableConcept               `bson:"topic,omitempty" json:"topic,omitempty"`
	Contributor     []Contributor                   `bson:"contributor,omitempty" json:"contributor,omitempty"`
	Contact         []ContactDetail                 `bson:"contact,omitempty" json:"contact,omitempty"`
	Copyright       string                          `bson:"copyright,omitempty" json:"copyright,omitempty"`
	RelatedArtifact []RelatedArtifact               `bson:"relatedArtifact,omitempty" json:"relatedArtifact,omitempty"`
	Library         []Reference                     `bson:"library,omitempty" json:"library,omitempty"`
	Goal            []PlanDefinitionGoalComponent   `bson:"goal,omitempty" json:"goal,omitempty"`
	Action          []PlanDefinitionActionComponent `bson:"action,omitempty" json:"action,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *PlanDefinition) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "PlanDefinition"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to PlanDefinition), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *PlanDefinition) GetBSON() (interface{}, error) {
	x.ResourceType = "PlanDefinition"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "planDefinition" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type planDefinition PlanDefinition

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *PlanDefinition) UnmarshalJSON(data []byte) (err error) {
	x2 := planDefinition{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = PlanDefinition(x2)
		return x.checkResourceType()
	}
	return
}

func (x *PlanDefinition) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "PlanDefinition"
	} else if x.ResourceType != "PlanDefinition" {
		return errors.New(fmt.Sprintf("Expected resourceType to be PlanDefinition, instead received %s", x.ResourceType))
	}
	return nil
}

type PlanDefinitionGoalComponent struct {
	BackboneElement `bson:",inline"`
	Category        *CodeableConcept                    `bson:"category,omitempty" json:"category,omitempty"`
	Description     *CodeableConcept                    `bson:"description,omitempty" json:"description,omitempty"`
	Priority        *CodeableConcept                    `bson:"priority,omitempty" json:"priority,omitempty"`
	Start           *CodeableConcept                    `bson:"start,omitempty" json:"start,omitempty"`
	Addresses       []CodeableConcept                   `bson:"addresses,omitempty" json:"addresses,omitempty"`
	Documentation   []RelatedArtifact                   `bson:"documentation,omitempty" json:"documentation,omitempty"`
	Target          []PlanDefinitionGoalTargetComponent `bson:"target,omitempty" json:"target,omitempty"`
}

type PlanDefinitionGoalTargetComponent struct {
	BackboneElement       `bson:",inline"`
	Measure               *CodeableConcept `bson:"measure,omitempty" json:"measure,omitempty"`
	DetailQuantity        *Quantity        `bson:"detailQuantity,omitempty" json:"detailQuantity,omitempty"`
	DetailRange           *Range           `bson:"detailRange,omitempty" json:"detailRange,omitempty"`
	DetailCodeableConcept *CodeableConcept `bson:"detailCodeableConcept,omitempty" json:"detailCodeableConcept,omitempty"`
	Due                   *Quantity        `bson:"due,omitempty" json:"due,omitempty"`
}

type PlanDefinitionActionComponent struct {
	BackboneElement     `bson:",inline"`
	Label               string                                       `bson:"label,omitempty" json:"label,omitempty"`
	Title               string                                       `bson:"title,omitempty" json:"title,omitempty"`
	Description         string                                       `bson:"description,omitempty" json:"description,omitempty"`
	TextEquivalent      string                                       `bson:"textEquivalent,omitempty" json:"textEquivalent,omitempty"`
	Code                []CodeableConcept                            `bson:"code,omitempty" json:"code,omitempty"`
	Reason              []CodeableConcept                            `bson:"reason,omitempty" json:"reason,omitempty"`
	Documentation       []RelatedArtifact                            `bson:"documentation,omitempty" json:"documentation,omitempty"`
	GoalId              []string                                     `bson:"goalId,omitempty" json:"goalId,omitempty"`
	TriggerDefinition   []TriggerDefinition                          `bson:"triggerDefinition,omitempty" json:"triggerDefinition,omitempty"`
	Condition           []PlanDefinitionActionConditionComponent     `bson:"condition,omitempty" json:"condition,omitempty"`
	Input               []DataRequirement                            `bson:"input,omitempty" json:"input,omitempty"`
	Output              []DataRequirement                            `bson:"output,omitempty" json:"output,omitempty"`
	RelatedAction       []PlanDefinitionActionRelatedActionComponent `bson:"relatedAction,omitempty" json:"relatedAction,omitempty"`
	TimingDateTime      *FHIRDateTime                                `bson:"timingDateTime,omitempty" json:"timingDateTime,omitempty"`
	TimingPeriod        *Period                                      `bson:"timingPeriod,omitempty" json:"timingPeriod,omitempty"`
	TimingDuration      *Quantity                                    `bson:"timingDuration,omitempty" json:"timingDuration,omitempty"`
	TimingRange         *Range                                       `bson:"timingRange,omitempty" json:"timingRange,omitempty"`
	TimingTiming        *Timing                                      `bson:"timingTiming,omitempty" json:"timingTiming,omitempty"`
	Participant         []PlanDefinitionActionParticipantComponent   `bson:"participant,omitempty" json:"participant,omitempty"`
	Type                *Coding                                      `bson:"type,omitempty" json:"type,omitempty"`
	GroupingBehavior    string                                       `bson:"groupingBehavior,omitempty" json:"groupingBehavior,omitempty"`
	SelectionBehavior   string                                       `bson:"selectionBehavior,omitempty" json:"selectionBehavior,omitempty"`
	RequiredBehavior    string                                       `bson:"requiredBehavior,omitempty" json:"requiredBehavior,omitempty"`
	PrecheckBehavior    string                                       `bson:"precheckBehavior,omitempty" json:"precheckBehavior,omitempty"`
	CardinalityBehavior string                                       `bson:"cardinalityBehavior,omitempty" json:"cardinalityBehavior,omitempty"`
	Definition          *Reference                                   `bson:"definition,omitempty" json:"definition,omitempty"`
	Transform           *Reference                                   `bson:"transform,omitempty" json:"transform,omitempty"`
	DynamicValue        []PlanDefinitionActionDynamicValueComponent  `bson:"dynamicValue,omitempty" json:"dynamicValue,omitempty"`
	Action              []PlanDefinitionActionComponent              `bson:"action,omitempty" json:"action,omitempty"`
}

type PlanDefinitionActionConditionComponent struct {
	BackboneElement `bson:",inline"`
	Kind            string `bson:"kind,omitempty" json:"kind,omitempty"`
	Description     string `bson:"description,omitempty" json:"description,omitempty"`
	Language        string `bson:"language,omitempty" json:"language,omitempty"`
	Expression      string `bson:"expression,omitempty" json:"expression,omitempty"`
}

type PlanDefinitionActionRelatedActionComponent struct {
	BackboneElement `bson:",inline"`
	ActionId        string    `bson:"actionId,omitempty" json:"actionId,omitempty"`
	Relationship    string    `bson:"relationship,omitempty" json:"relationship,omitempty"`
	OffsetDuration  *Quantity `bson:"offsetDuration,omitempty" json:"offsetDuration,omitempty"`
	OffsetRange     *Range    `bson:"offsetRange,omitempty" json:"offsetRange,omitempty"`
}

type PlanDefinitionActionParticipantComponent struct {
	BackboneElement `bson:",inline"`
	Type            string           `bson:"type,omitempty" json:"type,omitempty"`
	Role            *CodeableConcept `bson:"role,omitempty" json:"role,omitempty"`
}

type PlanDefinitionActionDynamicValueComponent struct {
	BackboneElement `bson:",inline"`
	Description     string `bson:"description,omitempty" json:"description,omitempty"`
	Path            string `bson:"path,omitempty" json:"path,omitempty"`
	Language        string `bson:"language,omitempty" json:"language,omitempty"`
	Expression      string `bson:"expression,omitempty" json:"expression,omitempty"`
}
