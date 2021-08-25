package v2

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"
	"github.com/CMSgov/bcda-app/log"

	r4Datatypes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	r4Models "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/episode_of_care_go_proto"
)

var (
	admissionDt = regexp.MustCompile(`^(ADMISSION_DT)$`)
	dischargeDt = regexp.MustCompile(`^(DISCHARGE_DT)$`)
	diagnosis   = regexp.MustCompile(`^(U071)|(B9729)$`)
	episode     = regexp.MustCompile(`^(COVID19_EPISODE)$`)
	month       = regexp.MustCompile(`^(COVID19_MONTH(0[1-9]|1[0-2]))$`)
)

func covidEpisode(mbi string, keyValue []utils.KvPair, lastUpdated time.Time) *r4Models.EpisodeOfCare {
	covidEpisode := &r4Models.EpisodeOfCare{}
	covidEpisode.Id = &r4Datatypes.Id{Value: "example-id-episode"}
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
				log.API.Warnf("Could not parse date string:", err)
			}
			covidEpisode.Period.Start = fhirTimestamp

		// discharge date maps to period end
		case dischargeDt.MatchString(kv.Key):
			fhirTimestamp, err := stringToFhirDate(kv.Value)
			if err != nil {
				log.API.Warnf("Could not parse date string:", err)
			}
			covidEpisode.Period.End = fhirTimestamp

		// CovidFlag extension
		// one of 12 columns corresponding to a calendar month;
		// has a value of either 0 or 1 indicating that month meets
		// the criteria for a covid episode
		case month.MatchString(kv.Key):
			val, err := strconv.ParseInt(kv.Value, 10, 32)
			if err != nil {
				log.API.Warnf("Could convert string to int for {}: {}", kv.Value, err)
			}
			// this will hold both the flag and period sub-extensions
			// see http://alr.cms.gov/ig/StructureDefinition/ext-covidFlag for reference
			covidFlagExt := []*r4Datatypes.Extension{}

			// flag sub-extension
			subExtFlag := &r4Datatypes.Extension{}
			subExtFlag.Url = &r4Datatypes.Uri{Value: "flag"}
			subExtFlag.Value = &r4Datatypes.Extension_ValueX{
				Choice: &r4Datatypes.Extension_ValueX_Integer{
					Integer: &r4Datatypes.Integer{Value: int32(val)},
				},
			}

			// period sub-extension
			subExtMonth := &r4Datatypes.Extension{}
			subExtMonth.Url = &r4Datatypes.Uri{Value: "monthNum"}

			monthNum := getMonthNumFromHeader(kv.Key)

			subExtMonth.Value = &r4Datatypes.Extension_ValueX{
				Choice: &r4Datatypes.Extension_ValueX_Integer{
					Integer: &r4Datatypes.Integer{
						Value: monthNum,
					},
				},
			}

			covidFlagExt = append(covidFlagExt, subExtFlag, subExtMonth)

			// this is the CovidFlag extension, plus it's corresponding URL
			// see http://alr.cms.gov/ig/StructureDefinition/ext-covidFlag for reference
			fullExt := &r4Datatypes.Extension{}
			fullExt.Url = &r4Datatypes.Uri{
				Value: "http://alr.cms.gov/ig/StructureDefinition/ext-covidFlag",
			}
			fullExt.Extension = covidFlagExt

			covidEpisode.Extension = append(covidEpisode.Extension, fullExt)

		// CovidEpisod extension
		case episode.MatchString(kv.Key):
			val, err := strconv.ParseInt(kv.Value, 10, 32)
			if err != nil {
				log.API.Warnf("Could convert string to int for {}: {}", kv.Value, err)
			}

			ext := &r4Datatypes.Extension{}
			ext.Url = &r4Datatypes.Uri{Value: "http://alr.cms.gov/ig/StructureDefinition/ext-covidEpisode"}
			ext.Value = &r4Datatypes.Extension_ValueX{
				Choice: &r4Datatypes.Extension_ValueX_Integer{
					Integer: &r4Datatypes.Integer{Value: int32(val)},
				},
			}
			covidEpisode.Extension = append(covidEpisode.Extension, ext)

		// one of two diagnosis codes (B9729 or U071)
		case diagnosis.MatchString(kv.Key):
			// Data with a value of 0 should not be included in the FHIR resource
			if kv.Value != "0" {
				episodeDiagnosis := []*r4Models.EpisodeOfCare_Diagnosis{}
				diagnosis := &r4Models.EpisodeOfCare_Diagnosis{
					Condition: &r4Datatypes.Reference{
						Identifier: &r4Datatypes.Identifier{
							System: &r4Datatypes.Uri{
								Value: "http://hl7.org/fhir/sid/icd-10",
							},
							Value: &r4Datatypes.String{
								Value: kv.Key,
							},
						},
					},
				}
				episodeDiagnosis = append(episodeDiagnosis, diagnosis)
				covidEpisode.Diagnosis = episodeDiagnosis
			}
		}
	}
	return covidEpisode

}

func stringToFhirDate(timeString string) (*r4Datatypes.DateTime, error) {
	timestamp, err := time.Parse("01/02/2006", timeString)
	if err != nil {
		return nil, err
	}
	fhirTimestamp := &r4Datatypes.DateTime{
		ValueUs:   timestamp.UnixNano() / int64(time.Microsecond),
		Precision: r4Datatypes.DateTime_DAY,
	}
	return fhirTimestamp, nil
}

// extracts the month value as an integer from the ALR column header
func getMonthNumFromHeader(header string) (num int32) {
	split := strings.Split(header, "_")
	monthString := split[1]
	monthNum := monthString[len(monthString)-2:]
	month, _ := strconv.ParseInt(monthNum, 10, 32)

	return int32(month)
}
