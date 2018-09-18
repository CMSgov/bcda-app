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

type ImagingStudy struct {
	DomainResource     `bson:",inline"`
	Uid                string                        `bson:"uid,omitempty" json:"uid,omitempty"`
	Accession          *Identifier                   `bson:"accession,omitempty" json:"accession,omitempty"`
	Identifier         []Identifier                  `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Availability       string                        `bson:"availability,omitempty" json:"availability,omitempty"`
	ModalityList       []Coding                      `bson:"modalityList,omitempty" json:"modalityList,omitempty"`
	Patient            *Reference                    `bson:"patient,omitempty" json:"patient,omitempty"`
	Context            *Reference                    `bson:"context,omitempty" json:"context,omitempty"`
	Started            *FHIRDateTime                 `bson:"started,omitempty" json:"started,omitempty"`
	BasedOn            []Reference                   `bson:"basedOn,omitempty" json:"basedOn,omitempty"`
	Referrer           *Reference                    `bson:"referrer,omitempty" json:"referrer,omitempty"`
	Interpreter        []Reference                   `bson:"interpreter,omitempty" json:"interpreter,omitempty"`
	Endpoint           []Reference                   `bson:"endpoint,omitempty" json:"endpoint,omitempty"`
	NumberOfSeries     *uint32                       `bson:"numberOfSeries,omitempty" json:"numberOfSeries,omitempty"`
	NumberOfInstances  *uint32                       `bson:"numberOfInstances,omitempty" json:"numberOfInstances,omitempty"`
	ProcedureReference []Reference                   `bson:"procedureReference,omitempty" json:"procedureReference,omitempty"`
	ProcedureCode      []CodeableConcept             `bson:"procedureCode,omitempty" json:"procedureCode,omitempty"`
	Reason             *CodeableConcept              `bson:"reason,omitempty" json:"reason,omitempty"`
	Description        string                        `bson:"description,omitempty" json:"description,omitempty"`
	Series             []ImagingStudySeriesComponent `bson:"series,omitempty" json:"series,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *ImagingStudy) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "ImagingStudy"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to ImagingStudy), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *ImagingStudy) GetBSON() (interface{}, error) {
	x.ResourceType = "ImagingStudy"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "imagingStudy" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type imagingStudy ImagingStudy

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *ImagingStudy) UnmarshalJSON(data []byte) (err error) {
	x2 := imagingStudy{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = ImagingStudy(x2)
		return x.checkResourceType()
	}
	return
}

func (x *ImagingStudy) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "ImagingStudy"
	} else if x.ResourceType != "ImagingStudy" {
		return errors.New(fmt.Sprintf("Expected resourceType to be ImagingStudy, instead received %s", x.ResourceType))
	}
	return nil
}

type ImagingStudySeriesComponent struct {
	BackboneElement   `bson:",inline"`
	Uid               string                                `bson:"uid,omitempty" json:"uid,omitempty"`
	Number            *uint32                               `bson:"number,omitempty" json:"number,omitempty"`
	Modality          *Coding                               `bson:"modality,omitempty" json:"modality,omitempty"`
	Description       string                                `bson:"description,omitempty" json:"description,omitempty"`
	NumberOfInstances *uint32                               `bson:"numberOfInstances,omitempty" json:"numberOfInstances,omitempty"`
	Availability      string                                `bson:"availability,omitempty" json:"availability,omitempty"`
	Endpoint          []Reference                           `bson:"endpoint,omitempty" json:"endpoint,omitempty"`
	BodySite          *Coding                               `bson:"bodySite,omitempty" json:"bodySite,omitempty"`
	Laterality        *Coding                               `bson:"laterality,omitempty" json:"laterality,omitempty"`
	Started           *FHIRDateTime                         `bson:"started,omitempty" json:"started,omitempty"`
	Performer         []Reference                           `bson:"performer,omitempty" json:"performer,omitempty"`
	Instance          []ImagingStudySeriesInstanceComponent `bson:"instance,omitempty" json:"instance,omitempty"`
}

type ImagingStudySeriesInstanceComponent struct {
	BackboneElement `bson:",inline"`
	Uid             string  `bson:"uid,omitempty" json:"uid,omitempty"`
	Number          *uint32 `bson:"number,omitempty" json:"number,omitempty"`
	SopClass        string  `bson:"sopClass,omitempty" json:"sopClass,omitempty"`
	Title           string  `bson:"title,omitempty" json:"title,omitempty"`
}
