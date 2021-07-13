package v1

import (
	"regexp"
	"strconv"
	"strings"
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
	month       = regexp.MustCompile(`^(COVID19_MONTH(0[1-9]|1[0-2]))$`)
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

		// one of 12 columns corresponding to a calendar month
		case month.MatchString(kv.Key):
			val, err := strconv.ParseInt(kv.Value, 10, 32)
			if err != nil {
				return nil
			}

			// this will hold both the flag and period sub-extensions
			// see http://alr.cms.gov/ig/StructureDefinition/ext-covidFlag for reference
			covidFlagExt := []*fhirdatatypes.Extension{}

			// flag sub-extension
			extFlag := &fhirdatatypes.Extension{}
			extFlag.Url = &fhirdatatypes.Uri{Value: "flag"}
			extFlag.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_Integer{
					Integer: &fhirdatatypes.Integer{Value: int32(val)},
				},
			}

			// period sub-extension
			extPeriod := &fhirdatatypes.Extension{}
			extPeriod.Url = &fhirdatatypes.Uri{Value: "period"}

			// determine the period start and end dates based on the column header
			// (e.g. COVID19_MONTH10 outputs 10/01/2020, 10/31/2020)
			startDt, endDt := getFlagPeriodFromHeader(kv.Key)

			start, _ := stringToFhirDate(startDt)
			end, _ := stringToFhirDate(endDt)

			extPeriod.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_Period{
					Period: &fhirdatatypes.Period{
						Start: start,
						End:   end,
					},
				},
			}

			covidFlagExt = append(covidFlagExt, extFlag, extPeriod)

			fullExt := &fhirdatatypes.Extension{}
			fullExt.Url = &fhirdatatypes.Uri{
				Value: "http://alr.cms.gov/ig/StructureDefinition/ext-covidFlag",
			}
			fullExt.Extension = covidFlagExt

			ext := covidEpisode.Extension
			ext = append(ext, fullExt)

			covidEpisode.Extension = ext

			// covid episode extension
		case episode.MatchString(kv.Key):
			val, err := strconv.ParseInt(kv.Value, 10, 32)
			if err != nil {
				return nil
			}

			ext := &fhirdatatypes.Extension{}
			ext.Url = &fhirdatatypes.Uri{Value: "http://alr.cms.gov/ig/StructureDefinition/ext-covidEpisode"}
			ext.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_Integer{
					Integer: &fhirdatatypes.Integer{Value: int32(val)},
				},
			}
			covidEpisode.Extension = append(covidEpisode.Extension, ext)

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
	timestamp, err := time.Parse("01/02/2006", timeString)

	if err != nil {
		return nil, err
	}
	fhirTimestamp := &fhirdatatypes.DateTime{
		ValueUs:   timestamp.UnixNano() / int64(time.Microsecond),
		Precision: fhirdatatypes.DateTime_DAY,
	}
	return fhirTimestamp, nil
}

// determines the start and end dates for a month given the calendar number (01-12)
func getFlagPeriodFromHeader(header string) (startDt string, endDt string) {
	split := strings.Split(header, "_")
	monthString := split[1]
	monthNum := monthString[len(monthString)-2:]

	var endNum string
	switch monthNum {
	case "01", "03", "05", "07", "08", "10", "12":
		endNum = "31"
	case "04", "06", "09", "11":
		endNum = "30"
	case "02":
		endNum = "29"
	}

	startDate := monthNum + `/01/2020`
	endDate := monthNum + `/` + endNum + `/2020`

	return startDate, endDate
}
