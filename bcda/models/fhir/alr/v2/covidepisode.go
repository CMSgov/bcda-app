package v2

import (
	"fmt"
	"regexp"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"

	r4Datatypes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	r4Models "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/episode_of_care_go_proto"
)

var (
	admissionDt = regexp.MustCompile(`^(ADMISSION_DT)$`)
	dischargeDt = regexp.MustCompile(`^(DISCHARGE_DT)$`)
	covidDates  = regexp.MustCompile(`^((ADMISSION_DT)|(DISCHARGE_DT))$`)

	covidEpisodeP = regexp.MustCompile(``)
	covidMonth    = regexp.MustCompile(`^(COVID19_MONTH(0[1-9]|1[0-2])$`)
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
		// admission and discharge dates map to period start and end
		case covidDates.MatchString(kv.Key):
			timestamp, err := time.Parse("2006-01-02T15:04:05.000-07:00", kv.Value)
			if err != nil {
				continue
			}
			fhirTimestamp := &r4Datatypes.DateTime{
				ValueUs:   timestamp.UnixNano() / int64(time.Microsecond),
				Precision: r4Datatypes.DateTime_DAY,
			}

			if admissionDt.MatchString(kv.Key) {
				covidEpisode.Period.Start = fhirTimestamp
			} else if dischargeDt.MatchString(kv.Key) {
				covidEpisode.Period.End = fhirTimestamp
			}
		case covidMonth.MatchString(kv.Key):
			ext := &r4Datatypes.Extension{}
			ext.Url = &r4Datatypes.Uri{Value: kv.Key}
			ext.Value = &r4Datatypes.Extension_ValueX{
				Choice: &r4Datatypes.Extension_ValueX_StringValue{
					StringValue: &r4Datatypes.String{Value: kv.Value},
				},
			}
		case covidEpisodeP.MatchString(kv.Key):
			ext := &r4Datatypes.Extension{}
			ext.Url = &r4Datatypes.Uri{Value: kv.Key}
			ext.Value = &r4Datatypes.Extension_ValueX{
				Choice: &r4Datatypes.Extension_ValueX_StringValue{
					StringValue: &r4Datatypes.String{Value: kv.Value},
				},
			}
			case covidDiagnosis.MatchString(kv.Key)
		}

		// if any of the headers are ones that only do binary values

		// if the header is a date

		// loop thru headers;
		// assign the following:
		// diagnosis codes,
		// admission and discharge dts,
		// episode (ext),
		// months (ext),

		// an extension can be:
		// a covid flag
		// a covid episode (months)

		fmt.Println("kv.Key")
		fmt.Println(kv.Key)
		fmt.Println("kv.Value")
		fmt.Println(kv.Value)

		covidEpisode.Extension = append()
	}
	return covidEpisode

}
