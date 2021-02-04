package responseutils

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/CMSgov/bcda-app/conf"

	fhirmodels "github.com/eug48/fhir/models"
)

func CreateOpOutcome(severity, code, detailsCode, detailsDisplay string) *fhirmodels.OperationOutcome {
	fhirmodels.DisableOperationOutcomeDiagnosticsFileLine()
	oo := fhirmodels.CreateOpOutcome(severity, code, detailsCode, detailsDisplay)
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
	bbServer := conf.GetEnv("BB_SERVER_LOCATION")
	statement := &fhirmodels.CapabilityStatement{
		Status:       "active",
		Date:         &fhirmodels.FHIRDateTime{Time: reldate, Precision: fhirmodels.Date},
		Publisher:    "Centers for Medicare & Medicaid Services",
		Kind:         "instance",
		Instantiates: []string{bbServer + "/baseDstu3/metadata/", "http://hl7.org/fhir/uv/bulkdata/CapabilityStatement/bulk-data"},
		Software: &fhirmodels.CapabilityStatementSoftwareComponent{
			Name:        "Beneficiary Claims Data API",
			Version:     relversion,
			ReleaseDate: &fhirmodels.FHIRDateTime{Time: reldate, Precision: fhirmodels.Date},
		},
		Implementation: &fhirmodels.CapabilityStatementImplementationComponent{
			Description: "The Beneficiary Claims Data API (BCDA) enables Accountable Care Organizations (ACOs) participating in the Shared Savings Program to retrieve Medicare Part A, Part B, and Part D claims data for their prospectively assigned or assignable beneficiaries.",
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
								{Display: "OAuth", Code: "OAuth", System: "http://terminology.hl7.org/CodeSystem/restful-security-service"},
							},
							Text: "OAuth",
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
						Name: "patient-export",
						Definition: &fhirmodels.Reference{
							Reference: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/patient-export",
						},
					},
					{
						Name: "group-export",
						Definition: &fhirmodels.Reference{
							Reference: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/group-export",
						},
					},
				},
			},
		},
	}
	addOauthEndpointToStatement(statement, baseurl)
	return statement
}
func addOauthEndpointToStatement(statement *fhirmodels.CapabilityStatement, baseurl string) {
	securityComponent := statement.Rest[0].Security
	extension := []fhirmodels.Extension{
		{
			Url: "http://fhir-registry.smarthealthit.org/StructureDefinition/oauth-uris",
			Extension: []fhirmodels.Extension{
				{
					Url:      "token",
					ValueUri: baseurl + "/auth/token",
				},
			},
		},
	}
	securityComponent.Extension = extension
	statement.Rest[0].Security = securityComponent
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
