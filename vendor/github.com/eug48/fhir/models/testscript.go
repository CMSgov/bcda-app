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

type TestScript struct {
	DomainResource `bson:",inline"`
	Url            string                           `bson:"url,omitempty" json:"url,omitempty"`
	Identifier     *Identifier                      `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Version        string                           `bson:"version,omitempty" json:"version,omitempty"`
	Name           string                           `bson:"name,omitempty" json:"name,omitempty"`
	Title          string                           `bson:"title,omitempty" json:"title,omitempty"`
	Status         string                           `bson:"status,omitempty" json:"status,omitempty"`
	Experimental   *bool                            `bson:"experimental,omitempty" json:"experimental,omitempty"`
	Date           *FHIRDateTime                    `bson:"date,omitempty" json:"date,omitempty"`
	Publisher      string                           `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Contact        []ContactDetail                  `bson:"contact,omitempty" json:"contact,omitempty"`
	Description    string                           `bson:"description,omitempty" json:"description,omitempty"`
	UseContext     []UsageContext                   `bson:"useContext,omitempty" json:"useContext,omitempty"`
	Jurisdiction   []CodeableConcept                `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	Purpose        string                           `bson:"purpose,omitempty" json:"purpose,omitempty"`
	Copyright      string                           `bson:"copyright,omitempty" json:"copyright,omitempty"`
	Origin         []TestScriptOriginComponent      `bson:"origin,omitempty" json:"origin,omitempty"`
	Destination    []TestScriptDestinationComponent `bson:"destination,omitempty" json:"destination,omitempty"`
	Metadata       *TestScriptMetadataComponent     `bson:"metadata,omitempty" json:"metadata,omitempty"`
	Fixture        []TestScriptFixtureComponent     `bson:"fixture,omitempty" json:"fixture,omitempty"`
	Profile        []Reference                      `bson:"profile,omitempty" json:"profile,omitempty"`
	Variable       []TestScriptVariableComponent    `bson:"variable,omitempty" json:"variable,omitempty"`
	Rule           []TestScriptRuleComponent        `bson:"rule,omitempty" json:"rule,omitempty"`
	Ruleset        []TestScriptRulesetComponent     `bson:"ruleset,omitempty" json:"ruleset,omitempty"`
	Setup          *TestScriptSetupComponent        `bson:"setup,omitempty" json:"setup,omitempty"`
	Test           []TestScriptTestComponent        `bson:"test,omitempty" json:"test,omitempty"`
	Teardown       *TestScriptTeardownComponent     `bson:"teardown,omitempty" json:"teardown,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *TestScript) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "TestScript"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to TestScript), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *TestScript) GetBSON() (interface{}, error) {
	x.ResourceType = "TestScript"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "testScript" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type testScript TestScript

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *TestScript) UnmarshalJSON(data []byte) (err error) {
	x2 := testScript{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = TestScript(x2)
		return x.checkResourceType()
	}
	return
}

func (x *TestScript) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "TestScript"
	} else if x.ResourceType != "TestScript" {
		return errors.New(fmt.Sprintf("Expected resourceType to be TestScript, instead received %s", x.ResourceType))
	}
	return nil
}

type TestScriptOriginComponent struct {
	BackboneElement `bson:",inline"`
	Index           *int32  `bson:"index,omitempty" json:"index,omitempty"`
	Profile         *Coding `bson:"profile,omitempty" json:"profile,omitempty"`
}

type TestScriptDestinationComponent struct {
	BackboneElement `bson:",inline"`
	Index           *int32  `bson:"index,omitempty" json:"index,omitempty"`
	Profile         *Coding `bson:"profile,omitempty" json:"profile,omitempty"`
}

type TestScriptMetadataComponent struct {
	BackboneElement `bson:",inline"`
	Link            []TestScriptMetadataLinkComponent       `bson:"link,omitempty" json:"link,omitempty"`
	Capability      []TestScriptMetadataCapabilityComponent `bson:"capability,omitempty" json:"capability,omitempty"`
}

type TestScriptMetadataLinkComponent struct {
	BackboneElement `bson:",inline"`
	Url             string `bson:"url,omitempty" json:"url,omitempty"`
	Description     string `bson:"description,omitempty" json:"description,omitempty"`
}

type TestScriptMetadataCapabilityComponent struct {
	BackboneElement `bson:",inline"`
	Required        *bool      `bson:"required,omitempty" json:"required,omitempty"`
	Validated       *bool      `bson:"validated,omitempty" json:"validated,omitempty"`
	Description     string     `bson:"description,omitempty" json:"description,omitempty"`
	Origin          []int32    `bson:"origin,omitempty" json:"origin,omitempty"`
	Destination     *int32     `bson:"destination,omitempty" json:"destination,omitempty"`
	Link            []string   `bson:"link,omitempty" json:"link,omitempty"`
	Capabilities    *Reference `bson:"capabilities,omitempty" json:"capabilities,omitempty"`
}

type TestScriptFixtureComponent struct {
	BackboneElement `bson:",inline"`
	Autocreate      *bool      `bson:"autocreate,omitempty" json:"autocreate,omitempty"`
	Autodelete      *bool      `bson:"autodelete,omitempty" json:"autodelete,omitempty"`
	Resource        *Reference `bson:"resource,omitempty" json:"resource,omitempty"`
}

type TestScriptVariableComponent struct {
	BackboneElement `bson:",inline"`
	Name            string `bson:"name,omitempty" json:"name,omitempty"`
	DefaultValue    string `bson:"defaultValue,omitempty" json:"defaultValue,omitempty"`
	Description     string `bson:"description,omitempty" json:"description,omitempty"`
	Expression      string `bson:"expression,omitempty" json:"expression,omitempty"`
	HeaderField     string `bson:"headerField,omitempty" json:"headerField,omitempty"`
	Hint            string `bson:"hint,omitempty" json:"hint,omitempty"`
	Path            string `bson:"path,omitempty" json:"path,omitempty"`
	SourceId        string `bson:"sourceId,omitempty" json:"sourceId,omitempty"`
}

type TestScriptRuleComponent struct {
	BackboneElement `bson:",inline"`
	Resource        *Reference                     `bson:"resource,omitempty" json:"resource,omitempty"`
	Param           []TestScriptRuleParamComponent `bson:"param,omitempty" json:"param,omitempty"`
}

type TestScriptRuleParamComponent struct {
	BackboneElement `bson:",inline"`
	Name            string `bson:"name,omitempty" json:"name,omitempty"`
	Value           string `bson:"value,omitempty" json:"value,omitempty"`
}

type TestScriptRulesetComponent struct {
	BackboneElement `bson:",inline"`
	Resource        *Reference                       `bson:"resource,omitempty" json:"resource,omitempty"`
	Rule            []TestScriptRulesetRuleComponent `bson:"rule,omitempty" json:"rule,omitempty"`
}

type TestScriptRulesetRuleComponent struct {
	BackboneElement `bson:",inline"`
	RuleId          string                                `bson:"ruleId,omitempty" json:"ruleId,omitempty"`
	Param           []TestScriptRulesetRuleParamComponent `bson:"param,omitempty" json:"param,omitempty"`
}

type TestScriptRulesetRuleParamComponent struct {
	BackboneElement `bson:",inline"`
	Name            string `bson:"name,omitempty" json:"name,omitempty"`
	Value           string `bson:"value,omitempty" json:"value,omitempty"`
}

type TestScriptSetupComponent struct {
	BackboneElement `bson:",inline"`
	Action          []TestScriptSetupActionComponent `bson:"action,omitempty" json:"action,omitempty"`
}

type TestScriptSetupActionComponent struct {
	BackboneElement `bson:",inline"`
	Operation       *TestScriptSetupActionOperationComponent `bson:"operation,omitempty" json:"operation,omitempty"`
	Assert          *TestScriptSetupActionAssertComponent    `bson:"assert,omitempty" json:"assert,omitempty"`
}

type TestScriptSetupActionOperationComponent struct {
	BackboneElement  `bson:",inline"`
	Type             *Coding                                                `bson:"type,omitempty" json:"type,omitempty"`
	Resource         string                                                 `bson:"resource,omitempty" json:"resource,omitempty"`
	Label            string                                                 `bson:"label,omitempty" json:"label,omitempty"`
	Description      string                                                 `bson:"description,omitempty" json:"description,omitempty"`
	Accept           string                                                 `bson:"accept,omitempty" json:"accept,omitempty"`
	ContentType      string                                                 `bson:"contentType,omitempty" json:"contentType,omitempty"`
	Destination      *int32                                                 `bson:"destination,omitempty" json:"destination,omitempty"`
	EncodeRequestUrl *bool                                                  `bson:"encodeRequestUrl,omitempty" json:"encodeRequestUrl,omitempty"`
	Origin           *int32                                                 `bson:"origin,omitempty" json:"origin,omitempty"`
	Params           string                                                 `bson:"params,omitempty" json:"params,omitempty"`
	RequestHeader    []TestScriptSetupActionOperationRequestHeaderComponent `bson:"requestHeader,omitempty" json:"requestHeader,omitempty"`
	RequestId        string                                                 `bson:"requestId,omitempty" json:"requestId,omitempty"`
	ResponseId       string                                                 `bson:"responseId,omitempty" json:"responseId,omitempty"`
	SourceId         string                                                 `bson:"sourceId,omitempty" json:"sourceId,omitempty"`
	TargetId         string                                                 `bson:"targetId,omitempty" json:"targetId,omitempty"`
	Url              string                                                 `bson:"url,omitempty" json:"url,omitempty"`
}

type TestScriptSetupActionOperationRequestHeaderComponent struct {
	BackboneElement `bson:",inline"`
	Field           string `bson:"field,omitempty" json:"field,omitempty"`
	Value           string `bson:"value,omitempty" json:"value,omitempty"`
}

type TestScriptSetupActionAssertComponent struct {
	BackboneElement           `bson:",inline"`
	Label                     string                                  `bson:"label,omitempty" json:"label,omitempty"`
	Description               string                                  `bson:"description,omitempty" json:"description,omitempty"`
	Direction                 string                                  `bson:"direction,omitempty" json:"direction,omitempty"`
	CompareToSourceId         string                                  `bson:"compareToSourceId,omitempty" json:"compareToSourceId,omitempty"`
	CompareToSourceExpression string                                  `bson:"compareToSourceExpression,omitempty" json:"compareToSourceExpression,omitempty"`
	CompareToSourcePath       string                                  `bson:"compareToSourcePath,omitempty" json:"compareToSourcePath,omitempty"`
	ContentType               string                                  `bson:"contentType,omitempty" json:"contentType,omitempty"`
	Expression                string                                  `bson:"expression,omitempty" json:"expression,omitempty"`
	HeaderField               string                                  `bson:"headerField,omitempty" json:"headerField,omitempty"`
	MinimumId                 string                                  `bson:"minimumId,omitempty" json:"minimumId,omitempty"`
	NavigationLinks           *bool                                   `bson:"navigationLinks,omitempty" json:"navigationLinks,omitempty"`
	Operator                  string                                  `bson:"operator,omitempty" json:"operator,omitempty"`
	Path                      string                                  `bson:"path,omitempty" json:"path,omitempty"`
	RequestMethod             string                                  `bson:"requestMethod,omitempty" json:"requestMethod,omitempty"`
	RequestURL                string                                  `bson:"requestURL,omitempty" json:"requestURL,omitempty"`
	Resource                  string                                  `bson:"resource,omitempty" json:"resource,omitempty"`
	Response                  string                                  `bson:"response,omitempty" json:"response,omitempty"`
	ResponseCode              string                                  `bson:"responseCode,omitempty" json:"responseCode,omitempty"`
	Rule                      *TestScriptActionAssertRuleComponent    `bson:"rule,omitempty" json:"rule,omitempty"`
	Ruleset                   *TestScriptActionAssertRulesetComponent `bson:"ruleset,omitempty" json:"ruleset,omitempty"`
	SourceId                  string                                  `bson:"sourceId,omitempty" json:"sourceId,omitempty"`
	ValidateProfileId         string                                  `bson:"validateProfileId,omitempty" json:"validateProfileId,omitempty"`
	Value                     string                                  `bson:"value,omitempty" json:"value,omitempty"`
	WarningOnly               *bool                                   `bson:"warningOnly,omitempty" json:"warningOnly,omitempty"`
}

type TestScriptActionAssertRuleComponent struct {
	BackboneElement `bson:",inline"`
	RuleId          string                                     `bson:"ruleId,omitempty" json:"ruleId,omitempty"`
	Param           []TestScriptActionAssertRuleParamComponent `bson:"param,omitempty" json:"param,omitempty"`
}

type TestScriptActionAssertRuleParamComponent struct {
	BackboneElement `bson:",inline"`
	Name            string `bson:"name,omitempty" json:"name,omitempty"`
	Value           string `bson:"value,omitempty" json:"value,omitempty"`
}

type TestScriptActionAssertRulesetComponent struct {
	BackboneElement `bson:",inline"`
	RulesetId       string                                       `bson:"rulesetId,omitempty" json:"rulesetId,omitempty"`
	Rule            []TestScriptActionAssertRulesetRuleComponent `bson:"rule,omitempty" json:"rule,omitempty"`
}

type TestScriptActionAssertRulesetRuleComponent struct {
	BackboneElement `bson:",inline"`
	RuleId          string                                            `bson:"ruleId,omitempty" json:"ruleId,omitempty"`
	Param           []TestScriptActionAssertRulesetRuleParamComponent `bson:"param,omitempty" json:"param,omitempty"`
}

type TestScriptActionAssertRulesetRuleParamComponent struct {
	BackboneElement `bson:",inline"`
	Name            string `bson:"name,omitempty" json:"name,omitempty"`
	Value           string `bson:"value,omitempty" json:"value,omitempty"`
}

type TestScriptTestComponent struct {
	BackboneElement `bson:",inline"`
	Name            string                          `bson:"name,omitempty" json:"name,omitempty"`
	Description     string                          `bson:"description,omitempty" json:"description,omitempty"`
	Action          []TestScriptTestActionComponent `bson:"action,omitempty" json:"action,omitempty"`
}

type TestScriptTestActionComponent struct {
	BackboneElement `bson:",inline"`
	Operation       *TestScriptSetupActionOperationComponent `bson:"operation,omitempty" json:"operation,omitempty"`
	Assert          *TestScriptSetupActionAssertComponent    `bson:"assert,omitempty" json:"assert,omitempty"`
}

type TestScriptTeardownComponent struct {
	BackboneElement `bson:",inline"`
	Action          []TestScriptTeardownActionComponent `bson:"action,omitempty" json:"action,omitempty"`
}

type TestScriptTeardownActionComponent struct {
	BackboneElement `bson:",inline"`
	Operation       *TestScriptSetupActionOperationComponent `bson:"operation,omitempty" json:"operation,omitempty"`
}
