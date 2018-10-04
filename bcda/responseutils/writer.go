package responseutils

import (
	"encoding/json"
	fhirmodels "github.com/eug48/fhir/models"
	"net/http"
	"time"
)

func CreateOpOutcome(severity, code, detailsCode, detailsDisplay string) *fhirmodels.OperationOutcome {
	fhirmodels.DisableOperationOutcomeDiagnosticsFileLine()
	oo := fhirmodels.CreateOpOutcome(severity, code, "", detailsDisplay)
	return oo
}

func WriteError(outcome *fhirmodels.OperationOutcome, w http.ResponseWriter, code int) {
	outcomeJSON, _ := json.Marshal(outcome)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err := w.Write(outcomeJSON)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func CreateCapabilityStatement() *fhirmodels.CapabilityStatement {
	usecoors := true
	dt := time.Now()
	baseurl := "<host>"
	statement := &fhirmodels.CapabilityStatement{
		Status:       "active",
		Date:         &fhirmodels.FHIRDateTime{Time: dt, Precision: fhirmodels.Date},
		Publisher:    "Centers for Medicare & Medicaid Services",
		Kind:         "capability",
		Instantiates: []string{"https://api.bluebutton.cms.gov/v1/fhir/metadata"},
		Software: &fhirmodels.CapabilityStatementSoftwareComponent{
			Name:        "Beneficiary Claims Data API",
			Version:     "0.1",
			ReleaseDate: &fhirmodels.FHIRDateTime{Time: dt, Precision: fhirmodels.Date},
		},
		Implementation: &fhirmodels.CapabilityStatementImplementationComponent{
			Description: "",
			Url:         baseurl,
		},
		FhirVersion:   "3.0.1",
		AcceptUnknown: "extensions",
		Format:        []string{"application/json", "application/fhir+json"},
		Rest: []fhirmodels.CapabilityStatementRestComponent{
			{
				Mode: "server",
				Security: &fhirmodels.CapabilityStatementRestSecurityComponent{
					Cors: &usecoors,
					Service: []fhirmodels.CodeableConcept{
						{
							Coding: []fhirmodels.Coding{
								{Display: "OAuth", Code: "OAuth", System: "http://hl7.org/fhir/ValueSet/restful-security-service"},
							},
							Text: "OAuth",
						},
						{
							Coding: []fhirmodels.Coding{
								{Display: "SMART-on-FHIR", Code: "SMART-on-FHIR", System: "http://hl7.org/fhir/ValueSet/restful-security-service"},
							},
							Text: "SMART-on-FHIR",
						},
					},
				},
				Interaction: []fhirmodels.CapabilityStatementSystemInteractionComponent{
					{
						Code: "batch",
					},
					{
						Code: "search-system",
					},
				},
				Operation: []fhirmodels.CapabilityStatementRestOperationComponent{
					{
						Name: "export",
						Definition: &fhirmodels.Reference{
							Reference: baseurl + "/api/v1/Patient/$export",
							Type:      "Endpoint",
						},
					},
					{
						Name: "jobs",
						Definition: &fhirmodels.Reference{
							Reference: baseurl + "/api/v1/jobs/{jobId}",
							Type:      "Endpoint",
						},
					},
					{
						Name: "token",
						Definition: &fhirmodels.Reference{
							Reference: baseurl + "/api/v1/token",
							Type:      "Endpoint",
						},
					},
					{
						Name: "bb_metadata",
						Definition: &fhirmodels.Reference{
							Reference: baseurl + "/api/v1/bb_metadata",
							Type:      "Endpoint",
						},
					},
				},
			},
		},
	}

	return statement
}

func WriteCapabilityStatement(statement *fhirmodels.CapabilityStatement, w http.ResponseWriter) {
	statementJSON, _ := json.Marshal(statement)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(statementJSON)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
