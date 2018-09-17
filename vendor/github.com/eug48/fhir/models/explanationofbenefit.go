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

type ExplanationOfBenefit struct {
	DomainResource       `bson:",inline"`
	Identifier           []Identifier                                         `bson:"identifier,omitempty" json:"identifier,omitempty"`
	Status               string                                               `bson:"status,omitempty" json:"status,omitempty"`
	Type                 *CodeableConcept                                     `bson:"type,omitempty" json:"type,omitempty"`
	SubType              []CodeableConcept                                    `bson:"subType,omitempty" json:"subType,omitempty"`
	Patient              *Reference                                           `bson:"patient,omitempty" json:"patient,omitempty"`
	BillablePeriod       *Period                                              `bson:"billablePeriod,omitempty" json:"billablePeriod,omitempty"`
	Created              *FHIRDateTime                                        `bson:"created,omitempty" json:"created,omitempty"`
	Enterer              *Reference                                           `bson:"enterer,omitempty" json:"enterer,omitempty"`
	Insurer              *Reference                                           `bson:"insurer,omitempty" json:"insurer,omitempty"`
	Provider             *Reference                                           `bson:"provider,omitempty" json:"provider,omitempty"`
	Organization         *Reference                                           `bson:"organization,omitempty" json:"organization,omitempty"`
	Referral             *Reference                                           `bson:"referral,omitempty" json:"referral,omitempty"`
	Facility             *Reference                                           `bson:"facility,omitempty" json:"facility,omitempty"`
	Claim                *Reference                                           `bson:"claim,omitempty" json:"claim,omitempty"`
	ClaimResponse        *Reference                                           `bson:"claimResponse,omitempty" json:"claimResponse,omitempty"`
	Outcome              *CodeableConcept                                     `bson:"outcome,omitempty" json:"outcome,omitempty"`
	Disposition          string                                               `bson:"disposition,omitempty" json:"disposition,omitempty"`
	Related              []ExplanationOfBenefitRelatedClaimComponent          `bson:"related,omitempty" json:"related,omitempty"`
	Prescription         *Reference                                           `bson:"prescription,omitempty" json:"prescription,omitempty"`
	OriginalPrescription *Reference                                           `bson:"originalPrescription,omitempty" json:"originalPrescription,omitempty"`
	Payee                *ExplanationOfBenefitPayeeComponent                  `bson:"payee,omitempty" json:"payee,omitempty"`
	Information          []ExplanationOfBenefitSupportingInformationComponent `bson:"information,omitempty" json:"information,omitempty"`
	CareTeam             []ExplanationOfBenefitCareTeamComponent              `bson:"careTeam,omitempty" json:"careTeam,omitempty"`
	Diagnosis            []ExplanationOfBenefitDiagnosisComponent             `bson:"diagnosis,omitempty" json:"diagnosis,omitempty"`
	Procedure            []ExplanationOfBenefitProcedureComponent             `bson:"procedure,omitempty" json:"procedure,omitempty"`
	Precedence           *uint32                                              `bson:"precedence,omitempty" json:"precedence,omitempty"`
	Insurance            *ExplanationOfBenefitInsuranceComponent              `bson:"insurance,omitempty" json:"insurance,omitempty"`
	Accident             *ExplanationOfBenefitAccidentComponent               `bson:"accident,omitempty" json:"accident,omitempty"`
	EmploymentImpacted   *Period                                              `bson:"employmentImpacted,omitempty" json:"employmentImpacted,omitempty"`
	Hospitalization      *Period                                              `bson:"hospitalization,omitempty" json:"hospitalization,omitempty"`
	Item                 []ExplanationOfBenefitItemComponent                  `bson:"item,omitempty" json:"item,omitempty"`
	AddItem              []ExplanationOfBenefitAddedItemComponent             `bson:"addItem,omitempty" json:"addItem,omitempty"`
	TotalCost            *Quantity                                            `bson:"totalCost,omitempty" json:"totalCost,omitempty"`
	UnallocDeductable    *Quantity                                            `bson:"unallocDeductable,omitempty" json:"unallocDeductable,omitempty"`
	TotalBenefit         *Quantity                                            `bson:"totalBenefit,omitempty" json:"totalBenefit,omitempty"`
	Payment              *ExplanationOfBenefitPaymentComponent                `bson:"payment,omitempty" json:"payment,omitempty"`
	Form                 *CodeableConcept                                     `bson:"form,omitempty" json:"form,omitempty"`
	ProcessNote          []ExplanationOfBenefitNoteComponent                  `bson:"processNote,omitempty" json:"processNote,omitempty"`
	BenefitBalance       []ExplanationOfBenefitBenefitBalanceComponent        `bson:"benefitBalance,omitempty" json:"benefitBalance,omitempty"`
}

// Custom marshaller to add the resourceType property, as required by the specification
func (resource *ExplanationOfBenefit) MarshalJSON() ([]byte, error) {
	resource.ResourceType = "ExplanationOfBenefit"
	// Dereferencing the pointer to avoid infinite recursion.
	// Passing in plain old x (a pointer to ExplanationOfBenefit), would cause this same
	// MarshallJSON function to be called again
	return json.Marshal(*resource)
}

func (x *ExplanationOfBenefit) GetBSON() (interface{}, error) {
	x.ResourceType = "ExplanationOfBenefit"
	// See comment in MarshallJSON to see why we dereference
	return *x, nil
}

// The "explanationOfBenefit" sub-type is needed to avoid infinite recursion in UnmarshalJSON
type explanationOfBenefit ExplanationOfBenefit

// Custom unmarshaller to properly unmarshal embedded resources (represented as interface{})
func (x *ExplanationOfBenefit) UnmarshalJSON(data []byte) (err error) {
	x2 := explanationOfBenefit{}
	if err = json.Unmarshal(data, &x2); err == nil {
		if x2.Contained != nil {
			for i := range x2.Contained {
				x2.Contained[i], err = MapToResource(x2.Contained[i], true)
				if err != nil {
					return err
				}
			}
		}
		*x = ExplanationOfBenefit(x2)
		return x.checkResourceType()
	}
	return
}

func (x *ExplanationOfBenefit) checkResourceType() error {
	if x.ResourceType == "" {
		x.ResourceType = "ExplanationOfBenefit"
	} else if x.ResourceType != "ExplanationOfBenefit" {
		return errors.New(fmt.Sprintf("Expected resourceType to be ExplanationOfBenefit, instead received %s", x.ResourceType))
	}
	return nil
}

type ExplanationOfBenefitRelatedClaimComponent struct {
	BackboneElement `bson:",inline"`
	Claim           *Reference       `bson:"claim,omitempty" json:"claim,omitempty"`
	Relationship    *CodeableConcept `bson:"relationship,omitempty" json:"relationship,omitempty"`
	Reference       *Identifier      `bson:"reference,omitempty" json:"reference,omitempty"`
}

type ExplanationOfBenefitPayeeComponent struct {
	BackboneElement `bson:",inline"`
	Type            *CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	ResourceType    *CodeableConcept `bson:"resourceType,omitempty" json:"resourceType,omitempty"`
	Party           *Reference       `bson:"party,omitempty" json:"party,omitempty"`
}

type ExplanationOfBenefitSupportingInformationComponent struct {
	BackboneElement `bson:",inline"`
	Sequence        *uint32          `bson:"sequence,omitempty" json:"sequence,omitempty"`
	Category        *CodeableConcept `bson:"category,omitempty" json:"category,omitempty"`
	Code            *CodeableConcept `bson:"code,omitempty" json:"code,omitempty"`
	TimingDate      *FHIRDateTime    `bson:"timingDate,omitempty" json:"timingDate,omitempty"`
	TimingPeriod    *Period          `bson:"timingPeriod,omitempty" json:"timingPeriod,omitempty"`
	ValueString     string           `bson:"valueString,omitempty" json:"valueString,omitempty"`
	ValueQuantity   *Quantity        `bson:"valueQuantity,omitempty" json:"valueQuantity,omitempty"`
	ValueAttachment *Attachment      `bson:"valueAttachment,omitempty" json:"valueAttachment,omitempty"`
	ValueReference  *Reference       `bson:"valueReference,omitempty" json:"valueReference,omitempty"`
	Reason          *Coding          `bson:"reason,omitempty" json:"reason,omitempty"`
}

type ExplanationOfBenefitCareTeamComponent struct {
	BackboneElement `bson:",inline"`
	Sequence        *uint32          `bson:"sequence,omitempty" json:"sequence,omitempty"`
	Provider        *Reference       `bson:"provider,omitempty" json:"provider,omitempty"`
	Responsible     *bool            `bson:"responsible,omitempty" json:"responsible,omitempty"`
	Role            *CodeableConcept `bson:"role,omitempty" json:"role,omitempty"`
	Qualification   *CodeableConcept `bson:"qualification,omitempty" json:"qualification,omitempty"`
}

type ExplanationOfBenefitDiagnosisComponent struct {
	BackboneElement          `bson:",inline"`
	Sequence                 *uint32           `bson:"sequence,omitempty" json:"sequence,omitempty"`
	DiagnosisCodeableConcept *CodeableConcept  `bson:"diagnosisCodeableConcept,omitempty" json:"diagnosisCodeableConcept,omitempty"`
	DiagnosisReference       *Reference        `bson:"diagnosisReference,omitempty" json:"diagnosisReference,omitempty"`
	Type                     []CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	PackageCode              *CodeableConcept  `bson:"packageCode,omitempty" json:"packageCode,omitempty"`
}

type ExplanationOfBenefitProcedureComponent struct {
	BackboneElement          `bson:",inline"`
	Sequence                 *uint32          `bson:"sequence,omitempty" json:"sequence,omitempty"`
	Date                     *FHIRDateTime    `bson:"date,omitempty" json:"date,omitempty"`
	ProcedureCodeableConcept *CodeableConcept `bson:"procedureCodeableConcept,omitempty" json:"procedureCodeableConcept,omitempty"`
	ProcedureReference       *Reference       `bson:"procedureReference,omitempty" json:"procedureReference,omitempty"`
}

type ExplanationOfBenefitInsuranceComponent struct {
	BackboneElement `bson:",inline"`
	Coverage        *Reference `bson:"coverage,omitempty" json:"coverage,omitempty"`
	PreAuthRef      []string   `bson:"preAuthRef,omitempty" json:"preAuthRef,omitempty"`
}

type ExplanationOfBenefitAccidentComponent struct {
	BackboneElement   `bson:",inline"`
	Date              *FHIRDateTime    `bson:"date,omitempty" json:"date,omitempty"`
	Type              *CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	LocationAddress   *Address         `bson:"locationAddress,omitempty" json:"locationAddress,omitempty"`
	LocationReference *Reference       `bson:"locationReference,omitempty" json:"locationReference,omitempty"`
}

type ExplanationOfBenefitItemComponent struct {
	BackboneElement         `bson:",inline"`
	Sequence                *uint32                                     `bson:"sequence,omitempty" json:"sequence,omitempty"`
	CareTeamLinkId          []uint32                                    `bson:"careTeamLinkId,omitempty" json:"careTeamLinkId,omitempty"`
	DiagnosisLinkId         []uint32                                    `bson:"diagnosisLinkId,omitempty" json:"diagnosisLinkId,omitempty"`
	ProcedureLinkId         []uint32                                    `bson:"procedureLinkId,omitempty" json:"procedureLinkId,omitempty"`
	InformationLinkId       []uint32                                    `bson:"informationLinkId,omitempty" json:"informationLinkId,omitempty"`
	Revenue                 *CodeableConcept                            `bson:"revenue,omitempty" json:"revenue,omitempty"`
	Category                *CodeableConcept                            `bson:"category,omitempty" json:"category,omitempty"`
	Service                 *CodeableConcept                            `bson:"service,omitempty" json:"service,omitempty"`
	Modifier                []CodeableConcept                           `bson:"modifier,omitempty" json:"modifier,omitempty"`
	ProgramCode             []CodeableConcept                           `bson:"programCode,omitempty" json:"programCode,omitempty"`
	ServicedDate            *FHIRDateTime                               `bson:"servicedDate,omitempty" json:"servicedDate,omitempty"`
	ServicedPeriod          *Period                                     `bson:"servicedPeriod,omitempty" json:"servicedPeriod,omitempty"`
	LocationCodeableConcept *CodeableConcept                            `bson:"locationCodeableConcept,omitempty" json:"locationCodeableConcept,omitempty"`
	LocationAddress         *Address                                    `bson:"locationAddress,omitempty" json:"locationAddress,omitempty"`
	LocationReference       *Reference                                  `bson:"locationReference,omitempty" json:"locationReference,omitempty"`
	Quantity                *Quantity                                   `bson:"quantity,omitempty" json:"quantity,omitempty"`
	UnitPrice               *Quantity                                   `bson:"unitPrice,omitempty" json:"unitPrice,omitempty"`
	Factor                  *float64                                    `bson:"factor,omitempty" json:"factor,omitempty"`
	Net                     *Quantity                                   `bson:"net,omitempty" json:"net,omitempty"`
	Udi                     []Reference                                 `bson:"udi,omitempty" json:"udi,omitempty"`
	BodySite                *CodeableConcept                            `bson:"bodySite,omitempty" json:"bodySite,omitempty"`
	SubSite                 []CodeableConcept                           `bson:"subSite,omitempty" json:"subSite,omitempty"`
	Encounter               []Reference                                 `bson:"encounter,omitempty" json:"encounter,omitempty"`
	NoteNumber              []uint32                                    `bson:"noteNumber,omitempty" json:"noteNumber,omitempty"`
	Adjudication            []ExplanationOfBenefitAdjudicationComponent `bson:"adjudication,omitempty" json:"adjudication,omitempty"`
	Detail                  []ExplanationOfBenefitDetailComponent       `bson:"detail,omitempty" json:"detail,omitempty"`
}

type ExplanationOfBenefitAdjudicationComponent struct {
	BackboneElement `bson:",inline"`
	Category        *CodeableConcept `bson:"category,omitempty" json:"category,omitempty"`
	Reason          *CodeableConcept `bson:"reason,omitempty" json:"reason,omitempty"`
	Amount          *Quantity        `bson:"amount,omitempty" json:"amount,omitempty"`
	Value           *float64         `bson:"value,omitempty" json:"value,omitempty"`
}

type ExplanationOfBenefitDetailComponent struct {
	BackboneElement `bson:",inline"`
	Sequence        *uint32                                     `bson:"sequence,omitempty" json:"sequence,omitempty"`
	Type            *CodeableConcept                            `bson:"type,omitempty" json:"type,omitempty"`
	Revenue         *CodeableConcept                            `bson:"revenue,omitempty" json:"revenue,omitempty"`
	Category        *CodeableConcept                            `bson:"category,omitempty" json:"category,omitempty"`
	Service         *CodeableConcept                            `bson:"service,omitempty" json:"service,omitempty"`
	Modifier        []CodeableConcept                           `bson:"modifier,omitempty" json:"modifier,omitempty"`
	ProgramCode     []CodeableConcept                           `bson:"programCode,omitempty" json:"programCode,omitempty"`
	Quantity        *Quantity                                   `bson:"quantity,omitempty" json:"quantity,omitempty"`
	UnitPrice       *Quantity                                   `bson:"unitPrice,omitempty" json:"unitPrice,omitempty"`
	Factor          *float64                                    `bson:"factor,omitempty" json:"factor,omitempty"`
	Net             *Quantity                                   `bson:"net,omitempty" json:"net,omitempty"`
	Udi             []Reference                                 `bson:"udi,omitempty" json:"udi,omitempty"`
	NoteNumber      []uint32                                    `bson:"noteNumber,omitempty" json:"noteNumber,omitempty"`
	Adjudication    []ExplanationOfBenefitAdjudicationComponent `bson:"adjudication,omitempty" json:"adjudication,omitempty"`
	SubDetail       []ExplanationOfBenefitSubDetailComponent    `bson:"subDetail,omitempty" json:"subDetail,omitempty"`
}

type ExplanationOfBenefitSubDetailComponent struct {
	BackboneElement `bson:",inline"`
	Sequence        *uint32                                     `bson:"sequence,omitempty" json:"sequence,omitempty"`
	Type            *CodeableConcept                            `bson:"type,omitempty" json:"type,omitempty"`
	Revenue         *CodeableConcept                            `bson:"revenue,omitempty" json:"revenue,omitempty"`
	Category        *CodeableConcept                            `bson:"category,omitempty" json:"category,omitempty"`
	Service         *CodeableConcept                            `bson:"service,omitempty" json:"service,omitempty"`
	Modifier        []CodeableConcept                           `bson:"modifier,omitempty" json:"modifier,omitempty"`
	ProgramCode     []CodeableConcept                           `bson:"programCode,omitempty" json:"programCode,omitempty"`
	Quantity        *Quantity                                   `bson:"quantity,omitempty" json:"quantity,omitempty"`
	UnitPrice       *Quantity                                   `bson:"unitPrice,omitempty" json:"unitPrice,omitempty"`
	Factor          *float64                                    `bson:"factor,omitempty" json:"factor,omitempty"`
	Net             *Quantity                                   `bson:"net,omitempty" json:"net,omitempty"`
	Udi             []Reference                                 `bson:"udi,omitempty" json:"udi,omitempty"`
	NoteNumber      []uint32                                    `bson:"noteNumber,omitempty" json:"noteNumber,omitempty"`
	Adjudication    []ExplanationOfBenefitAdjudicationComponent `bson:"adjudication,omitempty" json:"adjudication,omitempty"`
}

type ExplanationOfBenefitAddedItemComponent struct {
	BackboneElement `bson:",inline"`
	SequenceLinkId  []uint32                                        `bson:"sequenceLinkId,omitempty" json:"sequenceLinkId,omitempty"`
	Revenue         *CodeableConcept                                `bson:"revenue,omitempty" json:"revenue,omitempty"`
	Category        *CodeableConcept                                `bson:"category,omitempty" json:"category,omitempty"`
	Service         *CodeableConcept                                `bson:"service,omitempty" json:"service,omitempty"`
	Modifier        []CodeableConcept                               `bson:"modifier,omitempty" json:"modifier,omitempty"`
	Fee             *Quantity                                       `bson:"fee,omitempty" json:"fee,omitempty"`
	NoteNumber      []uint32                                        `bson:"noteNumber,omitempty" json:"noteNumber,omitempty"`
	Adjudication    []ExplanationOfBenefitAdjudicationComponent     `bson:"adjudication,omitempty" json:"adjudication,omitempty"`
	Detail          []ExplanationOfBenefitAddedItemsDetailComponent `bson:"detail,omitempty" json:"detail,omitempty"`
}

type ExplanationOfBenefitAddedItemsDetailComponent struct {
	BackboneElement `bson:",inline"`
	Revenue         *CodeableConcept                            `bson:"revenue,omitempty" json:"revenue,omitempty"`
	Category        *CodeableConcept                            `bson:"category,omitempty" json:"category,omitempty"`
	Service         *CodeableConcept                            `bson:"service,omitempty" json:"service,omitempty"`
	Modifier        []CodeableConcept                           `bson:"modifier,omitempty" json:"modifier,omitempty"`
	Fee             *Quantity                                   `bson:"fee,omitempty" json:"fee,omitempty"`
	NoteNumber      []uint32                                    `bson:"noteNumber,omitempty" json:"noteNumber,omitempty"`
	Adjudication    []ExplanationOfBenefitAdjudicationComponent `bson:"adjudication,omitempty" json:"adjudication,omitempty"`
}

type ExplanationOfBenefitPaymentComponent struct {
	BackboneElement  `bson:",inline"`
	Type             *CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	Adjustment       *Quantity        `bson:"adjustment,omitempty" json:"adjustment,omitempty"`
	AdjustmentReason *CodeableConcept `bson:"adjustmentReason,omitempty" json:"adjustmentReason,omitempty"`
	Date             *FHIRDateTime    `bson:"date,omitempty" json:"date,omitempty"`
	Amount           *Quantity        `bson:"amount,omitempty" json:"amount,omitempty"`
	Identifier       *Identifier      `bson:"identifier,omitempty" json:"identifier,omitempty"`
}

type ExplanationOfBenefitNoteComponent struct {
	BackboneElement `bson:",inline"`
	Number          *uint32          `bson:"number,omitempty" json:"number,omitempty"`
	Type            *CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	Text            string           `bson:"text,omitempty" json:"text,omitempty"`
	Language        *CodeableConcept `bson:"language,omitempty" json:"language,omitempty"`
}

type ExplanationOfBenefitBenefitBalanceComponent struct {
	BackboneElement `bson:",inline"`
	Category        *CodeableConcept                       `bson:"category,omitempty" json:"category,omitempty"`
	SubCategory     *CodeableConcept                       `bson:"subCategory,omitempty" json:"subCategory,omitempty"`
	Excluded        *bool                                  `bson:"excluded,omitempty" json:"excluded,omitempty"`
	Name            string                                 `bson:"name,omitempty" json:"name,omitempty"`
	Description     string                                 `bson:"description,omitempty" json:"description,omitempty"`
	Network         *CodeableConcept                       `bson:"network,omitempty" json:"network,omitempty"`
	Unit            *CodeableConcept                       `bson:"unit,omitempty" json:"unit,omitempty"`
	Term            *CodeableConcept                       `bson:"term,omitempty" json:"term,omitempty"`
	Financial       []ExplanationOfBenefitBenefitComponent `bson:"financial,omitempty" json:"financial,omitempty"`
}

type ExplanationOfBenefitBenefitComponent struct {
	BackboneElement    `bson:",inline"`
	Type               *CodeableConcept `bson:"type,omitempty" json:"type,omitempty"`
	AllowedUnsignedInt *uint32          `bson:"allowedUnsignedInt,omitempty" json:"allowedUnsignedInt,omitempty"`
	AllowedString      string           `bson:"allowedString,omitempty" json:"allowedString,omitempty"`
	AllowedMoney       *Quantity        `bson:"allowedMoney,omitempty" json:"allowedMoney,omitempty"`
	UsedUnsignedInt    *uint32          `bson:"usedUnsignedInt,omitempty" json:"usedUnsignedInt,omitempty"`
	UsedMoney          *Quantity        `bson:"usedMoney,omitempty" json:"usedMoney,omitempty"`
}
