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

type DocumentManifest struct {
	DomainResource   `bson:",inline"`
	MasterIdentifier *Identifier                        `bson:"masterIdentifier,omitempty" json:"masterIdentifier,omitempty"`
	Identifier       []Identifier                       `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status           string                             `bson:"status,omitempty" json:"status,omitempty"`
	Type             *CodeableConcept                   `bson:"type,omitempty" json:"type,omitempty"`
	Subject          *Reference                         `bson:"subject,omitempty" json:"subject,omitempty"`
	Created          *FHIRDateTime                      `bson:"created,omitempty" json:"created,omitempty"`
	Author           []Reference                        `bson:"author,omitempty" json:"author,omitempty"`
	Recipient        []Reference                        `bson:"recipient,omitempty" json:"recipient,omitempty"`
	Source           string                             `bson:"source,omitempty" json:"source,omitempty"`
	Description      string                             `bson:"description,omitempty" json:"description,omitempty"`
	Content          []DocumentManifestContentComponent `bson:"content,omitempty" json:"content,omitempty"`
	Related          []DocumentManifestRelatedComponent `bson:"related,omitempty" json:"related,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *DocumentManifest) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "DocumentManifest"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to DocumentManifest), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *DocumentManifest) GetBSON() (interface{}, error) {
	x.ResourceType = "DocumentManifest"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "documentManifest" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type documentManifest DocumentManifest

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *DocumentManifest) UnmarshalJSON(data []byte) (err error) {
	x2 := documentManifest{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = DocumentManifest(x2)
		return x.checkResourceType()
	}
	return
}

func (x *DocumentManifest) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "DocumentManifest"
	} else if x.ResourceType != "DocumentManifest" {
		return errors.New(fmt.Sprintf("Expected resourceType to be DocumentManifest, instead received %s", x.ResourceType))
	}
	return nil
}

type DocumentManifestContentComponent struct {
	BackboneElement `bson:",inline"`
	PAttachment     *Attachment `bson:"pAttachment,omitempty" json:"pAttachment,omitempty"`
	PReference      *Reference  `bson:"pReference,omitempty" json:"pReference,omitempty"`
}

type DocumentManifestRelatedComponent struct {
	BackboneElement `bson:",inline"`
	Identifier      *Identifier `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Ref             *Reference  `bson:"ref,omitempty" json:"ref,omitempty"`
}
