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

type Device struct {
	DomainResource  `bson:",inline"`
	Identifier      []Identifier        `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Udi             *DeviceUdiComponent `bson:"udi,omitempty" json:"udi,omitempty"`
	Status          string              `bson:"status,omitempty" json:"status,omitempty"`
	Type            *CodeableConcept    `bson:"type,omitempty" json:"type,omitempty"`
	LotNumber       string              `bson:"lotNumber,omitempty" json:"lotNumber,omitempty"`
	Manufacturer    string              `bson:"manufacturer,omitempty" json:"manufacturer,omitempty"`
	ManufactureDate *FHIRDateTime       `bson:"manufactureDate,omitempty" json:"manufactureDate,omitempty"`
	ExpirationDate  *FHIRDateTime       `bson:"expirationDate,omitempty" json:"expirationDate,omitempty"`
	Model           string              `bson:"model,omitempty" json:"model,omitempty"`
	Version         string              `bson:"version,omitempty" json:"version,omitempty"`
	Patient         *Reference          `bson:"patient,omitempty" json:"patient,omitempty"`
	Owner           *Reference          `bson:"owner,omitempty" json:"owner,omitempty"`
	Contact         []ContactPoint      `bson:"contact,omitempty" json:"contact,omitempty"`
	Location        *Reference          `bson:"location,omitempty" json:"location,omitempty"`
	Url             string              `bson:"url,omitempty" json:"url,omitempty"`
	Note            []Annotation        `bson:"note,omitempty" json:"note,omitempty"`
	Safety          []CodeableConcept   `bson:"safety,omitempty" json:"safety,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Device) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Device"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Device), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Device) GetBSON() (interface{}, error) {
	x.ResourceType = "Device"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "device" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type device Device

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Device) UnmarshalJSON(data []byte) (err error) {
	x2 := device{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Device(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Device) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Device"
	} else if x.ResourceType != "Device" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Device, instead received %s", x.ResourceType))
	}
	return nil
}

type DeviceUdiComponent struct {
	BackboneElement  `bson:",inline"`
	DeviceIdentifier string `bson:"deviceIdentifier,omitempty" json:"deviceIdentifier,omitempty"`
	Name             string `bson:"name,omitempty" json:"name,omitempty"`
	Jurisdiction     string `bson:"jurisdiction,omitempty" json:"jurisdiction,omitempty"`
	CarrierHRF       string `bson:"carrierHRF,omitempty" json:"carrierHRF,omitempty"`
	CarrierAIDC      string `bson:"carrierAIDC,omitempty" json:"carrierAIDC,omitempty"`
	Issuer           string `bson:"issuer,omitempty" json:"issuer,omitempty"`
	EntryType        string `bson:"entryType,omitempty" json:"entryType,omitempty"`
}
