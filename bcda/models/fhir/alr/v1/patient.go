package v1

import (
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

// Version 1:

func patient(alr *models.Alr) *fhirmodels.Patient {
	p := &fhirmodels.Patient{}
	p.Name = []*fhirdatatypes.HumanName{{Given: []*fhirdatatypes.String{fhirString(alr.BeneFirstName)},
		Family: fhirString(alr.BeneLastName)}}
	p.Gender = getGender(alr.BeneSex)
	p.BirthDate = fhirDate(alr.BeneDOB)
	p.Deceased = getDeceased(alr.BeneDOD)
	p.Address = getAddress(alr.KeyValue)
	p.Identifier = getIdentifiers(alr)
	p.Id = &fhirdatatypes.Id{Id: &fhirdatatypes.String{
		Value: "example-id-patient",
	}}
	p.Meta = &fhirdatatypes.Meta{
		LastUpdated: &fhirdatatypes.Instant{
			Precision: fhirdatatypes.Instant_SECOND,
			ValueUs:   alr.Timestamp.UnixNano() / int64(time.Microsecond),
		},
		Profile: []*fhirdatatypes.Uri{
			{Value: "http://alr.cms.gov/ig/StructureDefinition/alr-Patient"}},
	}

	// Extensions...
	extention := []*fhirdatatypes.Extension{}
	// TIN
	master := alr.KeyValue["MASTER_ID"]
	val := alr.KeyValue["B_EM_LINE_CNT_T"]
	if val != "" {
		field := makeSecondaryExt("serviceCount", val)
		if master != "" && field != nil {
			extSlice := []*fhirdatatypes.Extension{}
			tin := makeMainExt("http://terminology.hl7.org/CodeSystem/v2-0203",
				"TAX", "TAX ID Number", val)
			extSlice = append(extSlice, tin, field)
			extention = append(extention, &fhirdatatypes.Extension{
				Extension: extSlice,
				Url:       &fhirdatatypes.Uri{Value: "http://alr.cms.gov/ig/St…tion/ext-serviceCountTIN"},
			})
		}
	}
	// CCN
	val = alr.KeyValue["REV_LINE_CNT"]
	if val != "" {
		field := makeSecondaryExt("serviceCount", val)
		if master != "" && field != nil {
			extSlice := []*fhirdatatypes.Extension{}
			ccn := makeMainExt("https://bluebutton.cms.g…rces/variables/prvdr_num",
				"CCN", "CCN number", val)
			extSlice = append(extSlice, ccn, field)
			extention = append(extention, &fhirdatatypes.Extension{
				Extension: extSlice,
				Url:       &fhirdatatypes.Uri{Value: "http://alr.cms.gov/ig/St…tion/ext-serviceCountCCN"},
			})
		}
	}
	// TIN-NPI
	npi := alr.KeyValue["NPI_USED"]
	val = alr.KeyValue["PCS_COUNT"]
	if val != "" {
		field := makeSecondaryExt("serviceCount", val)
		if master != "" && field != nil {
			extSlice := []*fhirdatatypes.Extension{}
			tin := makeMainExt("http://terminology.hl7.org/CodeSystem/v2-0203",
				"TAX", "TAX ID Number", val)
			tinNPI := makeMainExt("http://hl7.org/fhir/sid/us-npi",
				"NPI", "NPI Number", npi)
			extSlice = append(extSlice, tin, tinNPI, field)
			extention = append(extention, &fhirdatatypes.Extension{
				Extension: extSlice,
				Url:       &fhirdatatypes.Uri{Value: "http://alr.cms.gov/ig/St…tion/ext-serviceCountTIN"},
			})
		}
	}

	p.Extension = extention

	return p
}

func makeMainExt(system, code, display, value string) *fhirdatatypes.Extension {

	ext := &fhirdatatypes.Extension{
		Value: &fhirdatatypes.Extension_ValueX{
			Choice: &fhirdatatypes.Extension_ValueX_Identifier{
				Identifier: &fhirdatatypes.Identifier{
					Type: &fhirdatatypes.CodeableConcept{
						Coding: []*fhirdatatypes.Coding{{
							System:  &fhirdatatypes.Uri{Value: system},
							Code:    &fhirdatatypes.Code{Value: code},
							Display: &fhirdatatypes.String{Value: display},
						}},
					},
					Value: &fhirdatatypes.String{Value: value},
				},
			},
		},
		Url: &fhirdatatypes.Uri{Value: "participant"},
	}

	return ext
}

func makeSecondaryExt(url, value string) *fhirdatatypes.Extension {
	val, err := strconv.ParseInt(value, 10, 32)

	if err != nil {
		return nil
	}

	ext := &fhirdatatypes.Extension{
		Url: &fhirdatatypes.Uri{Value: url},
		Value: &fhirdatatypes.Extension_ValueX{
			Choice: &fhirdatatypes.Extension_ValueX_Integer{
				Integer: &fhirdatatypes.Integer{Value: int32(val)},
			},
		},
	}

	return ext
}

func getGender(gender string) *fhircodes.AdministrativeGenderCode {
	code := &fhircodes.AdministrativeGenderCode{}
	switch gender {
	case "0":
		code.Value = fhircodes.AdministrativeGenderCode_UNKNOWN
	case "1":
		code.Value = fhircodes.AdministrativeGenderCode_MALE
	case "2":
		code.Value = fhircodes.AdministrativeGenderCode_FEMALE
	default:
		return nil
	}

	return code
}

func getDeceased(death time.Time) *fhirmodels.Patient_Deceased {
	if death.IsZero() {
		return nil
	}

	return &fhirmodels.Patient_Deceased{
		Deceased: &fhirmodels.Patient_Deceased_DateTime{DateTime: fhirDateTime(death)}}
}

func getAddress(kv map[string]string) []*fhirdatatypes.Address {
	address := &fhirdatatypes.Address{}
	address.State = fhirString(kv["GEO_SSA_STATE_NAME"])
	address.District = fhirString(kv["GEO_SSA_CNTY_CD_NAME"])

	if val := kv["STATE_COUNTY_CD"]; len(val) > 0 {
		address.Extension = []*fhirdatatypes.Extension{
			{Url: fhirURI("https://hl7.org/fhir/STU3/valueset-fips-county.html"),
				Value: &fhirdatatypes.Extension_ValueX{Choice: &fhirdatatypes.Extension_ValueX_StringValue{
					StringValue: fhirString(val),
				}},
			},
		}
	}
	return []*fhirdatatypes.Address{address}
}

func getIdentifiers(alr *models.Alr) []*fhirdatatypes.Identifier {
	var ids []*fhirdatatypes.Identifier
	ids = append(ids, mbiIdentifier(alr.BeneMBI))
	if len(alr.BeneHIC) > 0 {
		ids = append(ids, &fhirdatatypes.Identifier{
			System: fhirURI("http://hl7.org/fhir/sid/us-hicn"),
			Value:  fhirString(alr.BeneHIC),
		})
	}
	if val := alr.KeyValue["VA_TIN"]; len(val) > 0 {
		ids = append(ids, &fhirdatatypes.Identifier{
			System: fhirURI("http://hl7.org/fhir/sid/us-tin"),
			Value:  fhirString(val),
		})
	}
	if val := alr.KeyValue["VA_NPI"]; len(val) > 0 {
		ids = append(ids, &fhirdatatypes.Identifier{
			System: fhirURI("http://hl7.org/fhir/sid/us-npi"),
			Value:  fhirString(val),
		})
	}
	return ids
}
