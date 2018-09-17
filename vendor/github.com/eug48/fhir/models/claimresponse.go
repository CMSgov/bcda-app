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

type ClaimResponse struct {
	DomainResource       `bson:",inline"`
	Identifier           []Identifier                      `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status               string                            `bson:"status,omitempty" json:"status,omitempty"`
	Patient              *Reference                        `bson:"patient,omitempty" json:"patient,omitempty"`
	Created              *FHIRDateTime                     `bson:"created,omitempty" json:"created,omitempty"`
	Insurer              *Reference                        `bson:"insurer,omitempty" json:"insurer,omitempty"`
	RequestProvider      *Reference                        `bson:"requestProvider,omitempty" json:"requestProvider,omitempty"`
	RequestOrganization  *Reference                        `bson:"requestOrganization,omitempty" json:"requestOrganization,omitempty"`
	Request              *Reference                        `bson:"request,omitempty" json:"request,omitempty"`
	Outcome              *CodeableConcept                  `bson:"outcome,omitempty" json:"outcome,omitempty"`
	Disposition          string                            `bson:"disposition,omitempty" json:"disposition,omitempty"`
	PayeeType            *CodeableConcept                  `bson:"payeeType,omitempty" json:"payeeType,omitempty"`
	Item                 []ClaimResponseItemComponent      `bson:"item,omitempty" json:"item,omitempty"`
	AddItem              []ClaimResponseAddedItemComponent `bson:"addItem,omitempty" json:"addItem,omitempty"`
	Error                []ClaimResponseErrorComponent     `bson:"error,omitempty" json:"error,omitempty"`
	TotalCost            *Quantity                         `bson:"totalCost,omitempty" json:"totalCost,omitempty"`
	UnallocDeductable    *Quantity                         `bson:"unallocDeductable,omitempty" json:"unallocDeductable,omitempty"`
	TotalBenefit         *Quantity                         `bson:"totalBenefit,omitempty" json:"totalBenefit,omitempty"`
	Payment              *ClaimResponsePaymentComponent    `bson:"payment,omitempty" json:"payment,omitempty"`
	Reserved             *Coding                           `bson:"reserved,omitempty" json:"reserved,omitempty"`
	Form                 *CodeableConcept                  `bson:"form,omitempty" json:"form,omitempty"`
	ProcessNote          []ClaimResponseNoteComponent      `bson:"processNote,omitempty" json:"processNote,omitempty"`
	CommunicationRequest []Reference                       `bson:"communicationRequest,omitempty" json:"communicationRequest,omitempty"`
	Insurance            []ClaimResponseInsuranceComponent `bson:"insurance,omitempty" json:"insurance,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *ClaimResponse) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "ClaimResponse"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to ClaimResponse), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *ClaimResponse) GetBSON() (interface{}, error) {
	x.ResourceType = "ClaimResponse"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "claimResponse" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type claimResponse ClaimResponse

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *ClaimResponse) UnmarshalJSON(data []byte) (err error) {
	x2 := claimResponse{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = ClaimResponse(x2)
		return x.checkResourceType()
	}
	return
}

func (x *ClaimResponse) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "ClaimResponse"
	} else if x.ResourceType != "ClaimResponse" {
		return errors.New(fmt.Sprintf("Expected resourceType to be ClaimResponse, instead received %s", x.ResourceType))
	}
	return nil
}

type ClaimResponseItemComponent struct {
	BackboneElement `bson:",inline"`
	SequenceLinkId  *uint32                              `bson:"sequenceLinkId,omitempty" json:"sequenceLinkId,omitempty"`
	NoteNumber      []uint32                             `bson:"noteNumber,omitempty" json:"noteNumber,omitempty"`
	Adjudication    []ClaimResponseAdjudicationComponent `bson:"adjudication,omitempty" json:"adjudication,omitempty"`
	Detail          []ClaimResponseItemDetailComponent   `bson:"detail,omitempty" json:"detail,omitempty"`
}

type ClaimResponseAdjudicationComponent struct {
	BackboneElement `bson:",inline"`
	Category        *CodeableConcept `bson:"category,omitempty" json:"category,omitempty"`
	Reason          *CodeableConcept `bson:"reason,omitempty" json:"reason,omitempty"`
	Amount          *Quantity        `bson:"amount,omitempty" json:"amount,omitempty"`
	Value           *float64         `bson:"value,omitempty" json:"value,omitempty"`
}

type ClaimResponseItemDetailComponent struct {
	BackboneElement `bson:",inline"`
	SequenceLinkId  *uint32                              `bson:"sequenceLinkId,omitempty" json:"sequenceLinkId,omitempty"`
	NoteNumber      []uint32                             `bson:"noteNumber,omitempty" json:"noteNumber,omitempty"`
	Adjudication    []ClaimResponseAdjudicationComponent `bson:"adjudication,omitempty" json:"adjudication,omitempty"`
	SubDetail       []ClaimResponseSubDetailComponent    `bson:"subDetail,omitempty" json:"subDetail,omitempty"`
}

type ClaimResponseSubDetailComponent struct {
	BackboneElement `bson:",inline"`
	SequenceLinkId  *uint32                              `bson:"sequenceLinkId,omitempty" json:"sequenceLinkId,omitempty"`
	NoteNumber      []uint32                             `bson:"noteNumber,omitempty" json:"noteNumber,omitempty"`
	Adjudication    []ClaimResponseAdjudicationComponent `bson:"adjudication,omitempty" json:"adjudication,omitempty"`
}

type ClaimResponseAddedItemComponent struct {
	BackboneElement `bson:",inline"`
	SequenceLinkId  []uint32                                 `bson:"sequenceLinkId,omitempty" json:"sequenceLinkId,omitempty"`
	Revenue         *CodeableConcept                         `bson:"revenue,omitempty" json:"revenue,omitempty"`
	Category        *CodeableConcept                         `bson:"category,omitempty" json:"category,omitempty"`
	Service         *CodeableConcept                         `bson:"service,omitempty" json:"service,omitempty"`
	Modifier        []CodeableConcept                        `bson:"modifier,omitempty" json:"modifier,omitempty"`
	Fee             *Quantity                                `bson:"fee,omitempty" json:"fee,omitempty"`
	NoteNumber      []uint32                                 `bson:"noteNumber,omitempty" json:"noteNumber,omitempty"`
	Adjudication    []ClaimResponseAdjudicationComponent     `bson:"adjudication,omitempty" json:"adjudication,omitempty"`
	Detail          []ClaimResponseAddedItemsDetailComponent `bson:"detail,omitempty" json:"detail,omitempty"`
}

type ClaimResponseAddedItemsDetailComponent struct {
	BackboneElement `bson:",inline"`
	Revenue         *CodeableConcept                     `bson:"revenue,omitempty" json:"revenue,omitempty"`
	Category        *CodeableConcept                     `bson:"category,omitempty" json:"category,omitempty"`
	Service         *CodeableConcept                     `bson:"service,omitempty" json:"service,omitempty"`
	Modifier        []CodeableConcept                    `bson:"modifier,omitempty" json:"modifier,omitempty"`
	Fee             *Quantity                            `bson:"fee,omitempty" json:"fee,omitempty"`
	NoteNumber      []uint32                             `bson:"noteNumber,omitempty" json:"noteNumber,omitempty"`
	Adjudication    []ClaimResponseAdjudicationComponent `bson:"adjudication,omitempty" json:"adjudication,omitempty"`
}

type ClaimResponseErrorComponent struct {
	BackboneElement         `bson:",inline"`
	SequenceLinkId          *uint32          `bson:"sequenceLinkId,omitempty" json:"sequenceLinkId,omitempty"`
	DetailSequenceLinkId    *uint32          `bson:"detailSequenceLinkId,omitempty" json:"detailSequenceLinkId,omitempty"`
	SubdetailSequenceLinkId *uint32          `bson:"subdetailSequenceLinkId,omitempty" json:"subdetailSequenceLinkId,omitempty"`
	Code                    *CodeableConcept `bson:"code,omitempty" json:"code,omitempty"`
}

type ClaimResponsePaymentComponent struct {
	BackboneElement  `bson:",inline"`
	Type             *CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	Adjustment       *Quantity        `bson:"adjustment,omitempty" json:"adjustment,omitempty"`
	AdjustmentReason *CodeableConcept `bson:"adjustmentReason,omitempty" json:"adjustmentReason,omitempty"`
	Date             *FHIRDateTime    `bson:"date,omitempty" json:"date,omitempty"`
	Amount           *Quantity        `bson:"amount,omitempty" json:"amount,omitempty"`
	Identifier       *Identifier      `bson:"identifier,omitempty" json:"identifier,omitempty"`
}

type ClaimResponseNoteComponent struct {
	BackboneElement `bson:",inline"`
	Number          *uint32          `bson:"number,omitempty" json:"number,omitempty"`
	Type            *CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	Text            string           `bson:"text,omitempty" json:"text,omitempty"`
	Language        *CodeableConcept `bson:"language,omitempty" json:"language,omitempty"`
}

type ClaimResponseInsuranceComponent struct {
	BackboneElement     `bson:",inline"`
	Sequence            *uint32    `bson:"sequence,omitempty" json:"sequence,omitempty"`
	Focal               *bool      `bson:"focal,omitempty" json:"focal,omitempty"`
	Coverage            *Reference `bson:"coverage,omitempty" json:"coverage,omitempty"`
	BusinessArrangement string     `bson:"businessArrangement,omitempty" json:"businessArrangement,omitempty"`
	PreAuthRef          []string   `bson:"preAuthRef,omitempty" json:"preAuthRef,omitempty"`
	ClaimResponse       *Reference `bson:"claimResponse,omitempty" json:"claimResponse,omitempty"`
}
