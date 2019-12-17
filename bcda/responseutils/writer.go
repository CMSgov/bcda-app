package responseutils

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	fhirmodels "github.com/eug48/fhir/models"
)

func CreateOpOutcome(severity, code, detailsCode, detailsDisplay string) *fhirmodels.OperationOutcome {
	fhirmodels.DisableOperationOutcomeDiagnosticsFileLine()
	oo := fhirmodels.CreateOpOutcome(severity, code, "", detailsDisplay)
	return oo
}

func WriteError(outcome *fhirmodels.OperationOutcome, w http.ResponseWriter, code int) {
	outcomeJSON, err := json.Marshal(outcome)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(outcomeJSON)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func CreateCapabilityStatement(reldate time.Time, relversion, baseurl string) *fhirmodels.CapabilityStatement {
	usecors := true
	bbServer := os.Getenv("BB_SERVER_LOCATION")
	statement := &fhirmodels.CapabilityStatement{
		Status:       "active",
		Date:         &fhirmodels.FHIRDateTime{Time: reldate, Precision: fhirmodels.Date},
		Publisher:    "Centers for Medicare & Medicaid Services",
		Kind:         "capability",
		Instantiates: []string{bbServer + "/baseDstu3/metadata/"},
		Software: &fhirmodels.CapabilityStatementSoftwareComponent{
			Name:        "Beneficiary Claims Data API",
			Version:     relversion,
			ReleaseDate: &fhirmodels.FHIRDateTime{Time: reldate, Precision: fhirmodels.Date},
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
					Cors: &usecors,
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
						Name: "export",
						Definition: &fhirmodels.Reference{
							Reference: baseurl + "/api/v1/Group/[id]/$export",
							Type:      "Endpoint",
						},
					},
					{
						Name: "jobs",
						Definition: &fhirmodels.Reference{
							Reference: baseurl + "/api/v1/jobs/[jobId]",
							Type:      "Endpoint",
						},
					},
					{
						Name: "metadata",
						Definition: &fhirmodels.Reference{
							Reference: baseurl + "/api/v1/metadata",
							Type:      "Endpoint",
						},
					},
					{
						Name: "version",
						Definition: &fhirmodels.Reference{
							Reference: baseurl + "/_version",
							Type:      "Endpoint",
						},
					},
					{
						Name: "data",
						Definition: &fhirmodels.Reference{
							Reference: baseurl + "/data/[jobID]/[random_UUID].ndjson",
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
	statementJSON, err := json.Marshal(statement)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(statementJSON)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
