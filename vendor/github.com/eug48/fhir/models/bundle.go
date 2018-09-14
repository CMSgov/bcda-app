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
	"net/http"
	"strings"
	"gopkg.in/mgo.v2/bson"
)

type Bundle struct {
	Resource   `bson:",inline"`
	Identifier *Identifier            `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Type       string                 `bson:"type,omitempty" json:"type,omitempty"`
	Total      *uint32                `bson:"total,omitempty" json:"total,omitempty"`
	Link       []BundleLinkComponent  `bson:"link,omitempty" json:"link,omitempty"`
	Entry      []BundleEntryComponent `bson:"entry,omitempty" json:"entry,omitempty"`
	Signature  *Signature             `bson:"signature,omitempty" json:"signature,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Bundle) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Bundle"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Bundle), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Bundle) GetBSON() (interface{}, error) {
	x.ResourceType = "Bundle"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "bundle" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type bundle Bundle

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Bundle) UnmarshalJSON(data []byte) (err error) {
	x2 := bundle{}
	if err = json.Unmarshal(data, &x2); err == nil {
		*x = Bundle(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Bundle) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Bundle"
	} else if x.ResourceType != "Bundle" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Bundle, instead received %s", x.ResourceType))
	}
	return nil
}

type BundleLinkComponent struct {
	BackboneElement `bson:",inline"`
	Relation        string `bson:"relation,omitempty" json:"relation,omitempty"`
	Url             string `bson:"url,omitempty" json:"url,omitempty"`
}

type BundleEntryComponent struct {
	BackboneElement `bson:",inline"`
	Link            []BundleLinkComponent         `bson:"link,omitempty" json:"link,omitempty"`
	FullUrl         string                        `bson:"fullUrl,omitempty" json:"fullUrl,omitempty"`
	Resource        interface{}                   `bson:"resource,omitempty" json:"resource,omitempty"`
	Search          *BundleEntrySearchComponent   `bson:"search,omitempty" json:"search,omitempty"`
	Request         *BundleEntryRequestComponent  `bson:"request,omitempty" json:"request,omitempty"`
	Response        *BundleEntryResponseComponent `bson:"response,omitempty" json:"response,omitempty"`
}

// The "bundleEntryComponent" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type bundleEntryComponent BundleEntryComponent

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *BundleEntryComponent) UnmarshalJSON(data []byte) (err error) {
	x2 := bundleEntryComponent{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Resource != nil {
			x2.Resource, err = MapToResource(x2.Resource, true)
			if err != nil {
				return err
			}
		}
		*x = BundleEntryComponent(x2)
	}
	return
}

// Custom SetBSON implementation to properly deserialize embedded resources
// otherwise represented as interface{} into resource-specific structs as they
// are retrieved from the database.
func (x *BundleEntryComponent) SetBSON(raw bson.Raw) (err error) {
	x2 := bundleEntryComponent{}
	if err = raw.Unmarshal(&x2); err == nil {
		if x2.Resource != nil {
			x2.Resource, err = BSONMapToResource(x2.Resource.(bson.M), true)
			if err != nil {
				return err
			}
		}
		*x = BundleEntryComponent(x2)
	}
	return
}

type BundleEntrySearchComponent struct {
	BackboneElement `bson:",inline"`
	Mode            string   `bson:"mode,omitempty" json:"mode,omitempty"`
	Score           *float64 `bson:"score,omitempty" json:"score,omitempty"`
}

type BundleEntryRequestComponent struct {
	BackboneElement `bson:",inline"`
	Method          string        `bson:"method,omitempty" json:"method,omitempty"`
	Url             string        `bson:"url,omitempty" json:"url,omitempty"`
	IfNoneMatch     string        `bson:"ifNoneMatch,omitempty" json:"ifNoneMatch,omitempty"`
	IfModifiedSince *FHIRDateTime `bson:"ifModifiedSince,omitempty" json:"ifModifiedSince,omitempty"`
	IfMatch         string        `bson:"ifMatch,omitempty" json:"ifMatch,omitempty"`
	IfNoneExist     string        `bson:"ifNoneExist,omitempty" json:"ifNoneExist,omitempty"`
}

func (r *BundleEntryRequestComponent) DebugString() string {
	var str strings.Builder
	str.WriteString(fmt.Sprintf("%s %s", r.Method, r.Url))
	if r.IfNoneMatch != "" {
		str.WriteString(fmt.Sprintf(" | If-None-Match: %s", r.IfNoneMatch))
	}
	if r.IfModifiedSince != nil {
		str.WriteString(fmt.Sprintf(" | If-Modified-Since: %s", r.IfModifiedSince.Time.UTC().Format(http.TimeFormat)))
	}
	if r.IfMatch != "" {
		str.WriteString(fmt.Sprintf(" | If-Match: %s", r.IfMatch))
	}
	if r.IfNoneExist != "" {
		str.WriteString(fmt.Sprintf(" | If-None-Exist: %s", r.IfNoneExist))
	}
	return str.String()
}

type BundleEntryResponseComponent struct {
	BackboneElement `bson:",inline"`
	Status          string        `bson:"status,omitempty" json:"status,omitempty"`
	Location        string        `bson:"location,omitempty" json:"location,omitempty"`
	Etag            string        `bson:"etag,omitempty" json:"etag,omitempty"`
	LastModified    *FHIRDateTime `bson:"lastModified,omitempty" json:"lastModified,omitempty"`
	Outcome         interface{}   `bson:"outcome,omitempty" json:"outcome,omitempty"`
}

func (r *BundleEntryResponseComponent) DebugString() string {
	var str strings.Builder
	str.WriteString(fmt.Sprintf("%s", r.Status))
	if r.Location != "" {
		str.WriteString(fmt.Sprintf(" | Location: %s", r.Location))
	}
	if r.Etag != "" {
		str.WriteString(fmt.Sprintf(" | Etag: %s", r.Etag))
	}
	if r.LastModified != nil {
		str.WriteString(fmt.Sprintf(" | Last-Modified: %s", r.LastModified.Time.UTC().Format(http.TimeFormat)))
	}
	if r.Outcome != nil {
		str.WriteString(fmt.Sprintf(" | Outcome: %+v", r.Outcome))
	}
	return str.String()
}

// The "bundleEntryResponseComponent" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type bundleEntryResponseComponent BundleEntryResponseComponent

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *BundleEntryResponseComponent) UnmarshalJSON(data []byte) (err error) {
	x2 := bundleEntryResponseComponent{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Outcome != nil {
			x2.Outcome, err = MapToResource(x2.Outcome, true)
			if err != nil {
				return err
			}
		}
		*x = BundleEntryResponseComponent(x2)
	}
	return
}

// Custom SetBSON implementation to properly deserialize embedded resources
// otherwise represented as interface{} into resource-specific structs as they
// are retrieved from the database.
func (x *BundleEntryResponseComponent) SetBSON(raw bson.Raw) (err error) {
	x2 := bundleEntryResponseComponent{}
	if err = raw.Unmarshal(&x2); err == nil {
		if x2.Outcome != nil {
			x2.Outcome, err = BSONMapToResource(x2.Outcome.(bson.M), true)
			if err != nil {
				return err
			}
		}
		*x = BundleEntryResponseComponent(x2)
	}
	return
}
