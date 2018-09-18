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

type ExpansionProfile struct {
	DomainResource         `bson:",inline"`
	Url                    string                                   `bson:"url,omitempty" json:"url,omitempty"`
	Identifier             *Identifier                              `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Version                string                                   `bson:"version,omitempty" json:"version,omitempty"`
	Name                   string                                   `bson:"name,omitempty" json:"name,omitempty"`
	Status                 string                                   `bson:"status,omitempty" json:"status,omitempty"`
	Experimental           *bool                                    `bson:"experimental,omitempty" json:"experimental,omitempty"`
	Date                   *FHIRDateTime                            `bson:"date,omitempty" json:"date,omitempty"`
	Publisher              string                                   `bson:"publisher,omitempty" json:"publisher,omitempty"`
	Contact                []ContactDetail                          `bson:"contact,omitempty" json:"contact,omitempty"`
	Description            string                                   `bson:"description,omitempty" json:"description,omitempty"`
	UseContext             []UsageContext                           `bson:"useContext,omitempty" json:"useContext,omitempty"`
	Jurisdiction           []CodeableConcept                        `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	FixedVersion           []ExpansionProfileFixedVersionComponent  `bson:"fixedVersion,omitempty" json:"fixedVersion,omitempty"`
	ExcludedSystem         *ExpansionProfileExcludedSystemComponent `bson:"excludedSystem,omitempty" json:"excludedSystem,omitempty"`
	IncludeDesignations    *bool                                    `bson:"includeDesignations,omitempty" json:"includeDesignations,omitempty"`
	Designation            *ExpansionProfileDesignationComponent    `bson:"designation,omitempty" json:"designation,omitempty"`
	IncludeDefinition      *bool                                    `bson:"includeDefinition,omitempty" json:"includeDefinition,omitempty"`
	ActiveOnly             *bool                                    `bson:"activeOnly,omitempty" json:"activeOnly,omitempty"`
	ExcludeNested          *bool                                    `bson:"excludeNested,omitempty" json:"excludeNested,omitempty"`
	ExcludeNotForUI        *bool                                    `bson:"excludeNotForUI,omitempty" json:"excludeNotForUI,omitempty"`
	ExcludePostCoordinated *bool                                    `bson:"excludePostCoordinated,omitempty" json:"excludePostCoordinated,omitempty"`
	DisplayLanguage        string                                   `bson:"displayLanguage,omitempty" json:"displayLanguage,omitempty"`
	LimitedExpansion       *bool                                    `bson:"limitedExpansion,omitempty" json:"limitedExpansion,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *ExpansionProfile) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "ExpansionProfile"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to ExpansionProfile), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *ExpansionProfile) GetBSON() (interface{}, error) {
	x.ResourceType = "ExpansionProfile"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "expansionProfile" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type expansionProfile ExpansionProfile

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *ExpansionProfile) UnmarshalJSON(data []byte) (err error) {
	x2 := expansionProfile{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = ExpansionProfile(x2)
		return x.checkResourceType()
	}
	return
}

func (x *ExpansionProfile) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "ExpansionProfile"
	} else if x.ResourceType != "ExpansionProfile" {
		return errors.New(fmt.Sprintf("Expected resourceType to be ExpansionProfile, instead received %s", x.ResourceType))
	}
	return nil
}

type ExpansionProfileFixedVersionComponent struct {
	BackboneElement `bson:",inline"`
	System          string `bson:"system,omitempty" json:"system,omitempty"`
	Version         string `bson:"version,omitempty" json:"version,omitempty"`
	Mode            string `bson:"mode,omitempty" json:"mode,omitempty"`
}

type ExpansionProfileExcludedSystemComponent struct {
	BackboneElement `bson:",inline"`
	System          string `bson:"system,omitempty" json:"system,omitempty"`
	Version         string `bson:"version,omitempty" json:"version,omitempty"`
}

type ExpansionProfileDesignationComponent struct {
	BackboneElement `bson:",inline"`
	Include         *ExpansionProfileDesignationIncludeComponent `bson:"include,omitempty" json:"include,omitempty"`
	Exclude         *ExpansionProfileDesignationExcludeComponent `bson:"exclude,omitempty" json:"exclude,omitempty"`
}

type ExpansionProfileDesignationIncludeComponent struct {
	BackboneElement `bson:",inline"`
	Designation     []ExpansionProfileDesignationIncludeDesignationComponent `bson:"designation,omitempty" json:"designation,omitempty"`
}

type ExpansionProfileDesignationIncludeDesignationComponent struct {
	BackboneElement `bson:",inline"`
	Language        string  `bson:"language,omitempty" json:"language,omitempty"`
	Use             *Coding `bson:"use,omitempty" json:"use,omitempty"`
}

type ExpansionProfileDesignationExcludeComponent struct {
	BackboneElement `bson:",inline"`
	Designation     []ExpansionProfileDesignationExcludeDesignationComponent `bson:"designation,omitempty" json:"designation,omitempty"`
}

type ExpansionProfileDesignationExcludeDesignationComponent struct {
	BackboneElement `bson:",inline"`
	Language        string  `bson:"language,omitempty" json:"language,omitempty"`
	Use             *Coding `bson:"use,omitempty" json:"use,omitempty"`
}
