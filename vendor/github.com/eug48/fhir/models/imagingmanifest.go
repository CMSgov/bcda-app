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

type ImagingManifest struct {
	DomainResource `bson:",inline"`
	Identifier     *Identifier                     `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Patient        *Reference                      `bson:"patient,omitempty" json:"patient,omitempty"`
	AuthoringTime  *FHIRDateTime                   `bson:"authoringTime,omitempty" json:"authoringTime,omitempty"`
	Author         *Reference                      `bson:"author,omitempty" json:"author,omitempty"`
	Description    string                          `bson:"description,omitempty" json:"description,omitempty"`
	Study          []ImagingManifestStudyComponent `bson:"study,omitempty" json:"study,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *ImagingManifest) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "ImagingManifest"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to ImagingManifest), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *ImagingManifest) GetBSON() (interface{}, error) {
	x.ResourceType = "ImagingManifest"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "imagingManifest" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type imagingManifest ImagingManifest

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *ImagingManifest) UnmarshalJSON(data []byte) (err error) {
	x2 := imagingManifest{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = ImagingManifest(x2)
		return x.checkResourceType()
	}
	return
}

func (x *ImagingManifest) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "ImagingManifest"
	} else if x.ResourceType != "ImagingManifest" {
		return errors.New(fmt.Sprintf("Expected resourceType to be ImagingManifest, instead received %s", x.ResourceType))
	}
	return nil
}

type ImagingManifestStudyComponent struct {
	BackboneElement `bson:",inline"`
	Uid             string                           `bson:"uid,omitempty" json:"uid,omitempty"`
	ImagingStudy    *Reference                       `bson:"imagingStudy,omitempty" json:"imagingStudy,omitempty"`
	Endpoint        []Reference                      `bson:"endpoint,omitempty" json:"endpoint,omitempty"`
	Series          []ImagingManifestSeriesComponent `bson:"series,omitempty" json:"series,omitempty"`
}

type ImagingManifestSeriesComponent struct {
	BackboneElement `bson:",inline"`
	Uid             string                             `bson:"uid,omitempty" json:"uid,omitempty"`
	Endpoint        []Reference                        `bson:"endpoint,omitempty" json:"endpoint,omitempty"`
	Instance        []ImagingManifestInstanceComponent `bson:"instance,omitempty" json:"instance,omitempty"`
}

type ImagingManifestInstanceComponent struct {
	BackboneElement `bson:",inline"`
	SopClass        string `bson:"sopClass,omitempty" json:"sopClass,omitempty"`
	Uid             string `bson:"uid,omitempty" json:"uid,omitempty"`
}
