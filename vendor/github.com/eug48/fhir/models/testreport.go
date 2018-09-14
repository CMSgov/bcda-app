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

type TestReport struct {
	DomainResource `bson:",inline"`
	Identifier     *Identifier                      `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Name           string                           `bson:"name,omitempty" json:"name,omitempty"`
	Status         string                           `bson:"status,omitempty" json:"status,omitempty"`
	TestScript     *Reference                       `bson:"testScript,omitempty" json:"testScript,omitempty"`
	Result         string                           `bson:"result,omitempty" json:"result,omitempty"`
	Score          *float64                         `bson:"score,omitempty" json:"score,omitempty"`
	Tester         string                           `bson:"tester,omitempty" json:"tester,omitempty"`
	Issued         *FHIRDateTime                    `bson:"issued,omitempty" json:"issued,omitempty"`
	Participant    []TestReportParticipantComponent `bson:"participant,omitempty" json:"participant,omitempty"`
	Setup          *TestReportSetupComponent        `bson:"setup,omitempty" json:"setup,omitempty"`
	Test           []TestReportTestComponent        `bson:"test,omitempty" json:"test,omitempty"`
	Teardown       *TestReportTeardownComponent     `bson:"teardown,omitempty" json:"teardown,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *TestReport) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "TestReport"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to TestReport), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *TestReport) GetBSON() (interface{}, error) {
	x.ResourceType = "TestReport"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "testReport" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type testReport TestReport

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *TestReport) UnmarshalJSON(data []byte) (err error) {
	x2 := testReport{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = TestReport(x2)
		return x.checkResourceType()
	}
	return
}

func (x *TestReport) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "TestReport"
	} else if x.ResourceType != "TestReport" {
		return errors.New(fmt.Sprintf("Expected resourceType to be TestReport, instead received %s", x.ResourceType))
	}
	return nil
}

type TestReportParticipantComponent struct {
	BackboneElement `bson:",inline"`
	Type            string `bson:"type,omitempty" json:"type,omitempty"`
	Uri             string `bson:"uri,omitempty" json:"uri,omitempty"`
	Display         string `bson:"display,omitempty" json:"display,omitempty"`
}

type TestReportSetupComponent struct {
	BackboneElement `bson:",inline"`
	Action          []TestReportSetupActionComponent `bson:"action,omitempty" json:"action,omitempty"`
}

type TestReportSetupActionComponent struct {
	BackboneElement `bson:",inline"`
	Operation       *TestReportSetupActionOperationComponent `bson:"operation,omitempty" json:"operation,omitempty"`
	Assert          *TestReportSetupActionAssertComponent    `bson:"assert,omitempty" json:"assert,omitempty"`
}

type TestReportSetupActionOperationComponent struct {
	BackboneElement `bson:",inline"`
	Result          string `bson:"result,omitempty" json:"result,omitempty"`
	Message         string `bson:"message,omitempty" json:"message,omitempty"`
	Detail          string `bson:"detail,omitempty" json:"detail,omitempty"`
}

type TestReportSetupActionAssertComponent struct {
	BackboneElement `bson:",inline"`
	Result          string `bson:"result,omitempty" json:"result,omitempty"`
	Message         string `bson:"message,omitempty" json:"message,omitempty"`
	Detail          string `bson:"detail,omitempty" json:"detail,omitempty"`
}

type TestReportTestComponent struct {
	BackboneElement `bson:",inline"`
	Name            string                          `bson:"name,omitempty" json:"name,omitempty"`
	Description     string                          `bson:"description,omitempty" json:"description,omitempty"`
	Action          []TestReportTestActionComponent `bson:"action,omitempty" json:"action,omitempty"`
}

type TestReportTestActionComponent struct {
	BackboneElement `bson:",inline"`
	Operation       *TestReportSetupActionOperationComponent `bson:"operation,omitempty" json:"operation,omitempty"`
	Assert          *TestReportSetupActionAssertComponent    `bson:"assert,omitempty" json:"assert,omitempty"`
}

type TestReportTeardownComponent struct {
	BackboneElement `bson:",inline"`
	Action          []TestReportTeardownActionComponent `bson:"action,omitempty" json:"action,omitempty"`
}

type TestReportTeardownActionComponent struct {
	BackboneElement `bson:",inline"`
	Operation       *TestReportSetupActionOperationComponent `bson:"operation,omitempty" json:"operation,omitempty"`
}
