package responseutils

import (
	"encoding/json"
	fhirmodels "github.com/eug48/fhir/models"
	"net/http"
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

func WriteCapabilityStatement() {}
