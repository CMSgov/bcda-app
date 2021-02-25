package alr

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

func getPatient(alr models.Alr) *fhirmodels.Patient {
	p := &fhirmodels.Patient{}
	p.Name = []*fhirdatatypes.HumanName{{Given: []*fhirdatatypes.String{fhirString(alr.BeneFirstName)},
		Family: fhirString(alr.BeneLastName)}}
	p.Gender = getGender(alr.BeneSex)
	p.BirthDate = fhirDate(alr.BeneDOB)
	p.Deceased = getDeceased(alr.BeneDOD)
	p.Address = getAddress(alr.KeyValue)
	p.Identifier = getIdentifiers(alr)
	return p
}

func getGender(gender string) *fhircodes.AdministrativeGenderCode {
	if gender == "1" {
		return &fhircodes.AdministrativeGenderCode{Value: fhircodes.AdministrativeGenderCode_MALE}
	} else if gender == "2" {
		return &fhircodes.AdministrativeGenderCode{Value: fhircodes.AdministrativeGenderCode_FEMALE}
	} else {
		return nil
	}
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

func getIdentifiers(alr models.Alr) []*fhirdatatypes.Identifier {
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