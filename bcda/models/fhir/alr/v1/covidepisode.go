package v1

import (
	"regexp"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

var (
	admissionDt = regexp.MustCompile(`^(ADMISSION_DT)$`)
	dischargeDt = regexp.MustCompile(`^(DISCHARGE_DT)$`)
	diagnosis   = regexp.MustCompile(`^(U071)|(B9729)$`)
	episode     = regexp.MustCompile(`^(COVID19_EPISODE)$`)
	month       = regexp.MustCompile(`^(COVID19_MONTH(0[1-9]|1[0-2])$`)
)

func covidEpisode(mbi string, keyValue []utils.KvPair, lastUpdated time.Time) *fhirmodels.EpisodeOfCare {
	covidEpisode := &fhirmodels.EpisodeOfCare{}
	covidEpisode.Meta = &fhirdatatypes.Meta{
		LastUpdated: &fhirdatatypes.Instant{
			Precision: fhirdatatypes.Instant_SECOND,
			ValueUs:   lastUpdated.UnixNano() / int64(time.Microsecond),
		},
		Profile: []*fhirdatatypes.Uri{{
			Value: "http://alr.cms.gov/ig/StructureDefinition/alr-covidEpisode",
		}},
	}
	covidEpisode.Patient = &fhirdatatypes.Reference{
		Reference: &fhirdatatypes.Reference_PatientId{
			PatientId: &fhirdatatypes.ReferenceId{Value: mbi},
		},
	}
	covidEpisode.Period = &fhirdatatypes.Period{}
	covidEpisode.Extension = []*fhirdatatypes.Extension{}

	for _, kv := range keyValue {
		// FHIR does not include empty K:V pairs
		if kv.Value == "" {
			continue
		}

		switch {
		// admission date maps to period start
		case admissionDt.MatchString(kv.Key):
			fhirTimestamp, err := stringToFhirDate(kv.Value)
			if err != nil {
				continue
			}
			covidEpisode.Period.Start = fhirTimestamp

		// discharge date maps to period end
		case dischargeDt.MatchString(kv.Key):
			fhirTimestamp, err := stringToFhirDate(kv.Value)
			if err != nil {
				continue
			}
			covidEpisode.Period.End = fhirTimestamp

		// one of 12 flags corresponding to a calendar month
		case month.MatchString(kv.Key):
			ext := &fhirdatatypes.Extension{}
			ext.Url = &fhirdatatypes.Uri{Value: kv.Key}
			ext.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_StringValue{
					StringValue: &fhirdatatypes.String{Value: kv.Value},
				},
			}

		// covid episode extension
		case episode.MatchString(kv.Key):
			ext := &fhirdatatypes.Extension{}
			ext.Url = &fhirdatatypes.Uri{Value: kv.Key}
			ext.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_StringValue{
					StringValue: &fhirdatatypes.String{Value: kv.Value},
				},
			}
		// one of two diagnosis codes (B9729 or U071)
		case diagnosis.MatchString(kv.Key):
			episodeDiagnosis := []*fhirmodels.EpisodeOfCare_Diagnosis{}
			diagnosis := &fhirmodels.EpisodeOfCare_Diagnosis{
				Condition: &fhirdatatypes.Reference{
					Identifier: &fhirdatatypes.Identifier{
						System: &fhirdatatypes.Uri{
							Value: "http://hl7.org/fhir/sid/icd-10",
						},
						Value: &fhirdatatypes.String{
							Value: kv.Value,
						},
					},
				},
			}
			episodeDiagnosis = append(episodeDiagnosis, diagnosis)
			covidEpisode.Diagnosis = episodeDiagnosis
		}
	}
	return covidEpisode

}

func stringToFhirDate(timeString string) (*fhirdatatypes.DateTime, error) {
	timestamp, err := time.Parse("2006-01-02T15:04:05.000-07:00", timeString)
	if err != nil {
		return nil, err
	}
	fhirTimestamp := &fhirdatatypes.DateTime{
		ValueUs:   timestamp.UnixNano() / int64(time.Microsecond),
		Precision: fhirdatatypes.DateTime_DAY,
	}
	return fhirTimestamp, nil
}
