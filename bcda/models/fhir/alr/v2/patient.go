package v2

import (
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	r4Codes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/codes_go_proto"
	r4Datatypes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	r4Models "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/patient_go_proto"
)

// Version 2:
func patient(alr *models.Alr) *r4Models.Patient {
	p := &r4Models.Patient{}
	p.Name = []*r4Datatypes.HumanName{{
		Given:  []*r4Datatypes.String{{Value: alr.BeneFirstName}},
		Family: &r4Datatypes.String{Value: alr.BeneLastName},
	}}
	p.Gender = &r4Models.Patient_GenderCode{Value: getGenderV2(alr.BeneSex)}
	p.BirthDate = fhirDateV2(alr.BeneDOB)
	p.Deceased = getDeceasedV2(alr.BeneDOD)
	p.Address = getAddressV2(alr.KeyValue)
	p.Identifier = getIdentifiersV2(alr)
	p.Id = &r4Datatypes.Id{Id: &r4Datatypes.String{
		Value: "example-id-patient",
	}}
	p.Meta = &r4Datatypes.Meta{
		LastUpdated: &r4Datatypes.Instant{
			Precision: r4Datatypes.Instant_SECOND,
			ValueUs:   alr.Timestamp.UnixNano() / int64(time.Microsecond),
		},
		Profile: []*r4Datatypes.Canonical{
			{Value: "http://alr.cms.gov/ig/StructureDefinition/alr-Patient"}},
	}

	// Extensions...
	extention := []*r4Datatypes.Extension{}
	// TIN
	master := alr.KeyValue["MASTER_ID"]
	val := alr.KeyValue["B_EM_LINE_CNT_T"]
	if val != "" {
		field := makeSecondaryExtV2("serviceCount", val)
		if master != "" && field != nil {
			extSlice := []*r4Datatypes.Extension{}
			tin := makeMainExtV2("http://terminology.hl7.org/CodeSystem/v2-0203",
				"TAX", "TAX ID Number", val)
			extSlice = append(extSlice, tin, field)
			extention = append(extention, &r4Datatypes.Extension{
				Extension: extSlice,
				Url:       &r4Datatypes.Uri{Value: "http://alr.cms.gov/ig/St…tion/ext-serviceCountTIN"},
			})
		}
	}
	// CCN
	val = alr.KeyValue["REV_LINE_CNT"]
	if val != "" {
		field := makeSecondaryExtV2("serviceCount", val)
		if master != "" && field != nil {
			extSlice := []*r4Datatypes.Extension{}
			ccn := makeMainExtV2("https://bluebutton.cms.g…rces/variables/prvdr_num",
				"CCN", "CCN number", val)
			extSlice = append(extSlice, ccn, field)
			extention = append(extention, &r4Datatypes.Extension{
				Extension: extSlice,
				Url:       &r4Datatypes.Uri{Value: "http://alr.cms.gov/ig/St…tion/ext-serviceCountCCN"},
			})
		}
	}
	// TIN-NPI
	npi := alr.KeyValue["NPI_USED"]
	val = alr.KeyValue["PCS_COUNT"]
	if val != "" {
		field := makeSecondaryExtV2("serviceCount", val)
		if master != "" && field != nil {
			extSlice := []*r4Datatypes.Extension{}
			tin := makeMainExtV2("http://terminology.hl7.org/CodeSystem/v2-0203",
				"TAX", "TAX ID Number", val)
			tinNPI := makeMainExtV2("http://hl7.org/fhir/sid/us-npi",
				"NPI", "NPI Number", npi)
			extSlice = append(extSlice, tin, tinNPI, field)
			extention = append(extention, &r4Datatypes.Extension{
				Extension: extSlice,
				Url:       &r4Datatypes.Uri{Value: "http://alr.cms.gov/ig/St…tion/ext-serviceCountTIN"},
			})
		}
	}

	p.Extension = extention

	return p
}

func getGenderV2(gender string) r4Codes.AdministrativeGenderCode_Value {
	switch gender {
	case "0":
		return r4Codes.AdministrativeGenderCode_UNKNOWN
	case "1":
		return r4Codes.AdministrativeGenderCode_MALE
	case "2":
		return r4Codes.AdministrativeGenderCode_FEMALE
	default:
		return r4Codes.AdministrativeGenderCode_INVALID_UNINITIALIZED
	}
}

func fhirDateV2(t time.Time) *r4Datatypes.Date {
	micros := t.UnixNano() / int64(time.Microsecond)
	return &r4Datatypes.Date{ValueUs: micros, Precision: r4Datatypes.Date_DAY}
}

func getDeceasedV2(death time.Time) *r4Models.Patient_DeceasedX {
	if death.IsZero() {
		return nil
	}

	deceased := &r4Models.Patient_DeceasedX{}

	dateTime := &r4Datatypes.DateTime{
		ValueUs:   death.UnixNano() / int64(time.Microsecond),
		Precision: r4Datatypes.DateTime_DAY,
	}

	deceased.Choice = &r4Models.Patient_DeceasedX_DateTime{
		DateTime: dateTime,
	}

	return deceased
}

func getAddressV2(kv map[string]string) []*r4Datatypes.Address {
	address := &r4Datatypes.Address{}
	address.State = &r4Datatypes.String{Value: kv["GEO_SSA_STATE_NAME"]}
	address.District = &r4Datatypes.String{Value: kv["GEO_SSA_CNTY_CD_NAME"]}

	if val := kv["STATE_COUNTY_CD"]; len(val) > 0 {
		address.Extension = []*r4Datatypes.Extension{{
			Url: &r4Datatypes.Uri{Value: "https://hl7.org/fhir/STU3/valueset-fips-county.html"},
			Value: &r4Datatypes.Extension_ValueX{Choice: &r4Datatypes.Extension_ValueX_StringValue{
				StringValue: &r4Datatypes.String{Value: val},
			}},
		}}
	}
	return []*r4Datatypes.Address{address}
}

func getIdentifiersV2(alr *models.Alr) []*r4Datatypes.Identifier {
	var ids []*r4Datatypes.Identifier
	ids = append(ids, mbiIdentifierV2(alr.BeneMBI))
	if len(alr.BeneHIC) > 0 {
		ids = append(ids, &r4Datatypes.Identifier{
			System: &r4Datatypes.Uri{Value: "http://hl7.org/fhir/sid/us-hicn"},
			Value:  &r4Datatypes.String{Value: alr.BeneHIC},
		})
	}
	if val := alr.KeyValue["VA_TIN"]; len(val) > 0 {
		ids = append(ids, &r4Datatypes.Identifier{
			System: &r4Datatypes.Uri{Value: "http://hl7.org/fhir/sid/us-tin"},
			Value:  &r4Datatypes.String{Value: val},
		})
	}
	if val := alr.KeyValue["VA_NPI"]; len(val) > 0 {
		ids = append(ids, &r4Datatypes.Identifier{
			System: &r4Datatypes.Uri{Value: "http://hl7.org/fhir/sid/us-npi"},
			Value:  &r4Datatypes.String{Value: val},
		})
	}
	return ids
}

func mbiIdentifierV2(mbi string) *r4Datatypes.Identifier {
	return &r4Datatypes.Identifier{
		System: &r4Datatypes.Uri{Value: "http://hl7.org/fhir/sid/us-mbi"},
		Value:  &r4Datatypes.String{Value: mbi},
	}
}

func makeMainExtV2(system, code, display, value string) *r4Datatypes.Extension {

	ext := &r4Datatypes.Extension{
		Value: &r4Datatypes.Extension_ValueX{
			Choice: &r4Datatypes.Extension_ValueX_Identifier{
				Identifier: &r4Datatypes.Identifier{
					Type: &r4Datatypes.CodeableConcept{
						Coding: []*r4Datatypes.Coding{{
							System:  &r4Datatypes.Uri{Value: system},
							Code:    &r4Datatypes.Code{Value: code},
							Display: &r4Datatypes.String{Value: display},
						}},
					},
					Value: &r4Datatypes.String{Value: value},
				},
			},
		},
		Url: &r4Datatypes.Uri{Value: "participant"},
	}

	return ext
}

func makeSecondaryExtV2(url, value string) *r4Datatypes.Extension {
	val, err := strconv.ParseInt(value, 10, 32)

	if err != nil {
		return nil
	}

	ext := &r4Datatypes.Extension{
		Url: &r4Datatypes.Uri{Value: url},
		Value: &r4Datatypes.Extension_ValueX{
			Choice: &r4Datatypes.Extension_ValueX_Integer{
				Integer: &r4Datatypes.Integer{Value: int32(val)},
			},
		},
	}

	return ext
}
