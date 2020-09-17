package v2

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/samply/golang-fhir-models/fhir-models/fhir"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	log "github.com/sirupsen/logrus"
)

const (
	dateFormat = "2006-01-02"
)

/*
	swagger:route GET /api/v2/metadata metadata metadata

	Get metadata

	Returns metadata about the API.

	Produces:
	- application/fhir+json

	Schemes: http, https

	Responses:
		200: MetadataResponse
*/
func Metadata(w http.ResponseWriter, r *http.Request) {
	dt := time.Now()
	useCors := true
	bbServer := os.Getenv("BB_SERVER_LOCATION")

	scheme := "http"
	if servicemux.IsHTTPS(r) {
		scheme = "https"
	}
	host := fmt.Sprintf("%s://%s", scheme, r.Host)
	statement := fhir.CapabilityStatement{
		Status:    fhir.PublicationStatusActive,
		Date:      dt.Format(dateFormat),
		Publisher: getStringPtr("Centers for Medicare & Medicaid Services"),
		Kind:      fhir.CapabilityStatementKindInstance,
		// TODO (BCDA-3732): Update to r4 once endpoint is available
		Instantiates: []string{bbServer + "/baseDstu3/metadata/", "http://hl7.org/fhir/uv/bulkdata/CapabilityStatement/bulk-data"},
		Software: &fhir.CapabilityStatementSoftware{
			Name:        "Beneficiary Claims Data API",
			Version:     &constants.Version,
			ReleaseDate: getStringPtr(dt.Format(dateFormat)),
		},
		Implementation: &fhir.CapabilityStatementImplementation{
			Description: "The Beneficiary Claims Data API (BCDA) enables Accountable Care Organizations (ACOs) participating in the Shared Savings Program to retrieve Medicare Part A, Part B, and Part D claims data for their prospectively assigned or assignable beneficiaries.",
			Url:         &host,
		},
		FhirVersion: fhir.FHIRVersion4_0_1,
		Format:      []string{"application/json", "application/fhir+json"},
		Rest: []fhir.CapabilityStatementRest{
			{
				Mode: fhir.RestfulCapabilityModeServer,
				Security: &fhir.CapabilityStatementRestSecurity{
					Cors: &useCors,
					Service: []fhir.CodeableConcept{
						{
							Coding: []fhir.Coding{
								{
									Display: getStringPtr("OAuth"),
									Code:    getStringPtr("OAuth"),
									System:  getStringPtr("http://terminology.hl7.org/CodeSystem/restful-security-service"),
								},
							},
							Text: getStringPtr("OAuth"),
						},
					},
				},
				Interaction: []fhir.CapabilityStatementRestInteraction{
					{
						Code: fhir.SystemRestfulInteractionBatch,
					},
					{
						Code: fhir.SystemRestfulInteractionSearchSystem,
					},
				},
				Resource: []fhir.CapabilityStatementRestResource{
					{
						Type: fhir.ResourceTypePatient,
						Operation: []fhir.CapabilityStatementRestResourceOperation{
							{
								Name:       "patient-export",
								Definition: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/patient-export",
							},
						},
					},
					{
						Type: fhir.ResourceTypeGroup,
						Operation: []fhir.CapabilityStatementRestResourceOperation{
							{
								Name:       "group-export",
								Definition: "http://hl7.org/fhir/uv/bulkdata/OperationDefinition/group-export",
							},
						},
					},
				},
			},
		},
	}

	b, err := statement.MarshalJSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(b); err != nil {
		log.Errorf("Failed to write data %s", err.Error())
	}
}

func getStringPtr(value string) *string {
	return &value
}
