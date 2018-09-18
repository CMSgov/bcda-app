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

type RequestGroup struct {
	DomainResource        `bson:",inline"`
	Identifier            []Identifier                  `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Definition            []Reference                   `bson:"definition,omitempty" json:"definition,omitempty"`
	BasedOn               []Reference                   `bson:"basedOn,omitempty" json:"basedOn,omitempty"`
	Replaces              []Reference                   `bson:"replaces,omitempty" json:"replaces,omitempty"`
	GroupIdentifier       *Identifier                   `bson:"groupIdentifier,omitempty" json:"groupIdentifier,omitempty"`
	Status                string                        `bson:"status,omitempty" json:"status,omitempty"`
	Intent                string                        `bson:"intent,omitempty" json:"intent,omitempty"`
	Priority              string                        `bson:"priority,omitempty" json:"priority,omitempty"`
	Subject               *Reference                    `bson:"subject,omitempty" json:"subject,omitempty"`
	Context               *Reference                    `bson:"context,omitempty" json:"context,omitempty"`
	AuthoredOn            *FHIRDateTime                 `bson:"authoredOn,omitempty" json:"authoredOn,omitempty"`
	Author                *Reference                    `bson:"author,omitempty" json:"author,omitempty"`
	ReasonCodeableConcept *CodeableConcept              `bson:"reasonCodeableConcept,omitempty" json:"reasonCodeableConcept,omitempty"`
	ReasonReference       *Reference                    `bson:"reasonReference,omitempty" json:"reasonReference,omitempty"`
	Note                  []Annotation                  `bson:"note,omitempty" json:"note,omitempty"`
	Action                []RequestGroupActionComponent `bson:"action,omitempty" json:"action,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *RequestGroup) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "RequestGroup"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to RequestGroup), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *RequestGroup) GetBSON() (interface{}, error) {
	x.ResourceType = "RequestGroup"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "requestGroup" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type requestGroup RequestGroup

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *RequestGroup) UnmarshalJSON(data []byte) (err error) {
	x2 := requestGroup{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = RequestGroup(x2)
		return x.checkResourceType()
	}
	return
}

func (x *RequestGroup) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "RequestGroup"
	} else if x.ResourceType != "RequestGroup" {
		return errors.New(fmt.Sprintf("Expected resourceType to be RequestGroup, instead received %s", x.ResourceType))
	}
	return nil
}

type RequestGroupActionComponent struct {
	BackboneElement     `bson:",inline"`
	Label               string                                     `bson:"label,omitempty" json:"label,omitempty"`
	Title               string                                     `bson:"title,omitempty" json:"title,omitempty"`
	Description         string                                     `bson:"description,omitempty" json:"description,omitempty"`
	TextEquivalent      string                                     `bson:"textEquivalent,omitempty" json:"textEquivalent,omitempty"`
	Code                []CodeableConcept                          `bson:"code,omitempty" json:"code,omitempty"`
	Documentation       []RelatedArtifact                          `bson:"documentation,omitempty" json:"documentation,omitempty"`
	Condition           []RequestGroupActionConditionComponent     `bson:"condition,omitempty" json:"condition,omitempty"`
	RelatedAction       []RequestGroupActionRelatedActionComponent `bson:"relatedAction,omitempty" json:"relatedAction,omitempty"`
	TimingDateTime      *FHIRDateTime                              `bson:"timingDateTime,omitempty" json:"timingDateTime,omitempty"`
	TimingPeriod        *Period                                    `bson:"timingPeriod,omitempty" json:"timingPeriod,omitempty"`
	TimingDuration      *Quantity                                  `bson:"timingDuration,omitempty" json:"timingDuration,omitempty"`
	TimingRange         *Range                                     `bson:"timingRange,omitempty" json:"timingRange,omitempty"`
	TimingTiming        *Timing                                    `bson:"timingTiming,omitempty" json:"timingTiming,omitempty"`
	Participant         []Reference                                `bson:"participant,omitempty" json:"participant,omitempty"`
	Type                *Coding                                    `bson:"type,omitempty" json:"type,omitempty"`
	GroupingBehavior    string                                     `bson:"groupingBehavior,omitempty" json:"groupingBehavior,omitempty"`
	SelectionBehavior   string                                     `bson:"selectionBehavior,omitempty" json:"selectionBehavior,omitempty"`
	RequiredBehavior    string                                     `bson:"requiredBehavior,omitempty" json:"requiredBehavior,omitempty"`
	PrecheckBehavior    string                                     `bson:"precheckBehavior,omitempty" json:"precheckBehavior,omitempty"`
	CardinalityBehavior string                                     `bson:"cardinalityBehavior,omitempty" json:"cardinalityBehavior,omitempty"`
	Resource            *Reference                                 `bson:"resource,omitempty" json:"resource,omitempty"`
	Action              []RequestGroupActionComponent              `bson:"action,omitempty" json:"action,omitempty"`
}

type RequestGroupActionConditionComponent struct {
	BackboneElement `bson:",inline"`
	Kind            string `bson:"kind,omitempty" json:"kind,omitempty"`
	Description     string `bson:"description,omitempty" json:"description,omitempty"`
	Language        string `bson:"language,omitempty" json:"language,omitempty"`
	Expression      string `bson:"expression,omitempty" json:"expression,omitempty"`
}

type RequestGroupActionRelatedActionComponent struct {
	BackboneElement `bson:",inline"`
	ActionId        string    `bson:"actionId,omitempty" json:"actionId,omitempty"`
	Relationship    string    `bson:"relationship,omitempty" json:"relationship,omitempty"`
	OffsetDuration  *Quantity `bson:"offsetDuration,omitempty" json:"offsetDuration,omitempty"`
	OffsetRange     *Range    `bson:"offsetRange,omitempty" json:"offsetRange,omitempty"`
}
