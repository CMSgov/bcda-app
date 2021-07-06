package v2

import (
	"regexp"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"

	r4Datatypes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	r4Models "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/episode_of_care_go_proto"
)

var (
	admissionDt = regexp.MustCompile(`^(ADMISSION_DT)$`)
	dischargeDt = regexp.MustCompile(`^(DISCHARGE_DT)$`)
	diagnosis   = regexp.MustCompile(`^(U071)|(B9729)$`)
	episode     = regexp.MustCompile(`^(COVID19_EPISODE)$`)
	month       = regexp.MustCompile(`^(COVID19_MONTH(0[1-9]|1[0-2])$`)
)

func covidEpisode(mbi string, keyValue []utils.KvPair, lastUpdated time.Time) *r4Models.EpisodeOfCare {
	covidEpisode := &r4Models.EpisodeOfCare{}
	covidEpisode.Meta = &r4Datatypes.Meta{
		LastUpdated: &r4Datatypes.Instant{
			Precision: r4Datatypes.Instant_SECOND,
			ValueUs:   lastUpdated.UnixNano() / int64(time.Microsecond),
		},
		Profile: []*r4Datatypes.Canonical{{
			Value: "http://alr.cms.gov/ig/StructureDefinition/alr-covidEpisode",
		}},
	}
	covidEpisode.Patient = &r4Datatypes.Reference{
		Reference: &r4Datatypes.Reference_PatientId{
			PatientId: &r4Datatypes.ReferenceId{Value: mbi},
		},
	}
	covidEpisode.Period = &r4Datatypes.Period{}
	covidEpisode.Extension = []*r4Datatypes.Extension{}

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
			ext := &r4Datatypes.Extension{}
			ext.Url = &r4Datatypes.Uri{Value: kv.Key}
			ext.Value = &r4Datatypes.Extension_ValueX{
				Choice: &r4Datatypes.Extension_ValueX_StringValue{
					StringValue: &r4Datatypes.String{Value: kv.Value}, // does this need to be an int?
				},
			}

		// covid episode extension
		case episode.MatchString(kv.Key):
			ext := &r4Datatypes.Extension{}
			ext.Url = &r4Datatypes.Uri{Value: kv.Key}
			ext.Value = &r4Datatypes.Extension_ValueX{
				Choice: &r4Datatypes.Extension_ValueX_StringValue{
					StringValue: &r4Datatypes.String{Value: kv.Value},
				},
			}
		// one of two diagnosis codes (B9729 or U071)
		case diagnosis.MatchString(kv.Key):
			episodeDiagnosis := []*r4Models.EpisodeOfCare_Diagnosis{}
			diagnosis := &r4Models.EpisodeOfCare_Diagnosis{
				Condition: &r4Datatypes.Reference{
					Identifier: &r4Datatypes.Identifier{
						System: &r4Datatypes.Uri{
							Value: "http://hl7.org/fhir/sid/icd-10",
						},
						Value: &r4Datatypes.String{
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

func stringToFhirDate(timeString string) (*r4Datatypes.DateTime, error) {
	timestamp, err := time.Parse("2006-01-02T15:04:05.000-07:00", timeString)
	if err != nil {
		return nil, err
	}
	fhirTimestamp := &r4Datatypes.DateTime{
		ValueUs:   timestamp.UnixNano() / int64(time.Microsecond),
		Precision: r4Datatypes.DateTime_DAY,
	}
	return fhirTimestamp, nil
}
