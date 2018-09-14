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

type Sequence struct {
	DomainResource   `bson:",inline"`
	Identifier       []Identifier                   `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Type             string                         `bson:"type,omitempty" json:"type,omitempty"`
	CoordinateSystem *int32                         `bson:"coordinateSystem,omitempty" json:"coordinateSystem,omitempty"`
	Patient          *Reference                     `bson:"patient,omitempty" json:"patient,omitempty"`
	Specimen         *Reference                     `bson:"specimen,omitempty" json:"specimen,omitempty"`
	Device           *Reference                     `bson:"device,omitempty" json:"device,omitempty"`
	Performer        *Reference                     `bson:"performer,omitempty" json:"performer,omitempty"`
	Quantity         *Quantity                      `bson:"quantity,omitempty" json:"quantity,omitempty"`
	ReferenceSeq     *SequenceReferenceSeqComponent `bson:"referenceSeq,omitempty" json:"referenceSeq,omitempty"`
	Variant          []SequenceVariantComponent     `bson:"variant,omitempty" json:"variant,omitempty"`
	ObservedSeq      string                         `bson:"observedSeq,omitempty" json:"observedSeq,omitempty"`
	Quality          []SequenceQualityComponent     `bson:"quality,omitempty" json:"quality,omitempty"`
	ReadCoverage     *int32                         `bson:"readCoverage,omitempty" json:"readCoverage,omitempty"`
	Repository       []SequenceRepositoryComponent  `bson:"repository,omitempty" json:"repository,omitempty"`
	Pointer          []Reference                    `bson:"pointer,omitempty" json:"pointer,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *Sequence) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "Sequence"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to Sequence), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *Sequence) GetBSON() (interface{}, error) {
	x.ResourceType = "Sequence"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "sequence" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type sequence Sequence

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *Sequence) UnmarshalJSON(data []byte) (err error) {
	x2 := sequence{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = Sequence(x2)
		return x.checkResourceType()
	}
	return
}

func (x *Sequence) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "Sequence"
	} else if x.ResourceType != "Sequence" {
		return errors.New(fmt.Sprintf("Expected resourceType to be Sequence, instead received %s", x.ResourceType))
	}
	return nil
}

type SequenceReferenceSeqComponent struct {
	BackboneElement     `bson:",inline"`
	Chromosome          *CodeableConcept `bson:"chromosome,omitempty" json:"chromosome,omitempty"`
	GenomeBuild         string           `bson:"genomeBuild,omitempty" json:"genomeBuild,omitempty"`
	ReferenceSeqId      *CodeableConcept `bson:"referenceSeqId,omitempty" json:"referenceSeqId,omitempty"`
	ReferenceSeqPointer *Reference       `bson:"referenceSeqPointer,omitempty" json:"referenceSeqPointer,omitempty"`
	ReferenceSeqString  string           `bson:"referenceSeqString,omitempty" json:"referenceSeqString,omitempty"`
	Strand              *int32           `bson:"strand,omitempty" json:"strand,omitempty"`
	WindowStart         *int32           `bson:"windowStart,omitempty" json:"windowStart,omitempty"`
	WindowEnd           *int32           `bson:"windowEnd,omitempty" json:"windowEnd,omitempty"`
}

type SequenceVariantComponent struct {
	BackboneElement `bson:",inline"`
	Start           *int32     `bson:"start,omitempty" json:"start,omitempty"`
	End             *int32     `bson:"end,omitempty" json:"end,omitempty"`
	ObservedAllele  string     `bson:"observedAllele,omitempty" json:"observedAllele,omitempty"`
	ReferenceAllele string     `bson:"referenceAllele,omitempty" json:"referenceAllele,omitempty"`
	Cigar           string     `bson:"cigar,omitempty" json:"cigar,omitempty"`
	VariantPointer  *Reference `bson:"variantPointer,omitempty" json:"variantPointer,omitempty"`
}

type SequenceQualityComponent struct {
	BackboneElement  `bson:",inline"`
	Type             string           `bson:"type,omitempty" json:"type,omitempty"`
	StandardSequence *CodeableConcept `bson:"standardSequence,omitempty" json:"standardSequence,omitempty"`
	Start            *int32           `bson:"start,omitempty" json:"start,omitempty"`
	End              *int32           `bson:"end,omitempty" json:"end,omitempty"`
	Score            *Quantity        `bson:"score,omitempty" json:"score,omitempty"`
	Method           *CodeableConcept `bson:"method,omitempty" json:"method,omitempty"`
	TruthTP          *float64         `bson:"truthTP,omitempty" json:"truthTP,omitempty"`
	QueryTP          *float64         `bson:"queryTP,omitempty" json:"queryTP,omitempty"`
	TruthFN          *float64         `bson:"truthFN,omitempty" json:"truthFN,omitempty"`
	QueryFP          *float64         `bson:"queryFP,omitempty" json:"queryFP,omitempty"`
	GtFP             *float64         `bson:"gtFP,omitempty" json:"gtFP,omitempty"`
	Precision        *float64         `bson:"precision,omitempty" json:"precision,omitempty"`
	Recall           *float64         `bson:"recall,omitempty" json:"recall,omitempty"`
	FScore           *float64         `bson:"fScore,omitempty" json:"fScore,omitempty"`
}

type SequenceRepositoryComponent struct {
	BackboneElement `bson:",inline"`
	Type            string `bson:"type,omitempty" json:"type,omitempty"`
	Url             string `bson:"url,omitempty" json:"url,omitempty"`
	Name            string `bson:"name,omitempty" json:"name,omitempty"`
	DatasetId       string `bson:"datasetId,omitempty" json:"datasetId,omitempty"`
	VariantsetId    string `bson:"variantsetId,omitempty" json:"variantsetId,omitempty"`
	ReadsetId       string `bson:"readsetId,omitempty" json:"readsetId,omitempty"`
}
