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

type MeasureReport struct {
	DomainResource        `bson:",inline"`
	Identifier            *Identifier                   `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status                string                        `bson:"status,omitempty" json:"status,omitempty"`
	Type                  string                        `bson:"type,omitempty" json:"type,omitempty"`
	Measure               *Reference                    `bson:"measure,omitempty" json:"measure,omitempty"`
	Patient               *Reference                    `bson:"patient,omitempty" json:"patient,omitempty"`
	Date                  *FHIRDateTime                 `bson:"date,omitempty" json:"date,omitempty"`
	ReportingOrganization *Reference                    `bson:"reportingOrganization,omitempty" json:"reportingOrganization,omitempty"`
	Period                *Period                       `bson:"period,omitempty" json:"period,omitempty"`
	Group                 []MeasureReportGroupComponent `bson:"group,omitempty" json:"group,omitempty"`
	EvaluatedResources    *Reference                    `bson:"evaluatedResources,omitempty" json:"evaluatedResources,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *MeasureReport) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "MeasureReport"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to MeasureReport), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *MeasureReport) GetBSON() (interface{}, error) {
	x.ResourceType = "MeasureReport"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "measureReport" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type measureReport MeasureReport

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *MeasureReport) UnmarshalJSON(data []byte) (err error) {
	x2 := measureReport{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = MeasureReport(x2)
		return x.checkResourceType()
	}
	return
}

func (x *MeasureReport) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "MeasureReport"
	} else if x.ResourceType != "MeasureReport" {
		return errors.New(fmt.Sprintf("Expected resourceType to be MeasureReport, instead received %s", x.ResourceType))
	}
	return nil
}

type MeasureReportGroupComponent struct {
	BackboneElement `bson:",inline"`
	Identifier      *Identifier                             `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Population      []MeasureReportGroupPopulationComponent `bson:"population,omitempty" json:"population,omitempty"`
	MeasureScore    *float64                                `bson:"measureScore,omitempty" json:"measureScore,omitempty"`
	Stratifier      []MeasureReportGroupStratifierComponent `bson:"stratifier,omitempty" json:"stratifier,omitempty"`
}

type MeasureReportGroupPopulationComponent struct {
	BackboneElement `bson:",inline"`
	Identifier      *Identifier      `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Code            *CodeableConcept `bson:"code,omitempty" json:"code,omitempty"`
	Count           *int32           `bson:"count,omitempty" json:"count,omitempty"`
	Patients        *Reference       `bson:"patients,omitempty" json:"patients,omitempty"`
}

type MeasureReportGroupStratifierComponent struct {
	BackboneElement `bson:",inline"`
	Identifier      *Identifier                             `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Stratum         []MeasureReportStratifierGroupComponent `bson:"stratum,omitempty" json:"stratum,omitempty"`
}

type MeasureReportStratifierGroupComponent struct {
	BackboneElement `bson:",inline"`
	Value           string                                            `bson:"value,omitempty" json:"value,omitempty"`
	Population      []MeasureReportStratifierGroupPopulationComponent `bson:"population,omitempty" json:"population,omitempty"`
	MeasureScore    *float64                                          `bson:"measureScore,omitempty" json:"measureScore,omitempty"`
}

type MeasureReportStratifierGroupPopulationComponent struct {
	BackboneElement `bson:",inline"`
	Identifier      *Identifier      `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Code            *CodeableConcept `bson:"code,omitempty" json:"code,omitempty"`
	Count           *int32           `bson:"count,omitempty" json:"count,omitempty"`
	Patients        *Reference       `bson:"patients,omitempty" json:"patients,omitempty"`
}
