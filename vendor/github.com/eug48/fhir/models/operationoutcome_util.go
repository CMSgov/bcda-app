package models

import (
	"path"
	"fmt"
	"runtime"
)


var showOpOutcomeDiagnostics = true

func DisableOperationOutcomeDiagnosticsFileLine() { // e.g. for testing OperationOutcomes
	showOpOutcomeDiagnostics = false
}
func getOpOutcomeDiagnostics() string {
	if showOpOutcomeDiagnostics {
		_, file, line, _ := runtime.Caller(3)
		return fmt.Sprintf("%s:%d", path.Base(file), line)
	} else {
		return ""
	}
}

func CreateOpOutcome(severity, code, detailsCode, detailsDisplay string) *OperationOutcome {

	outcome := &OperationOutcome{
		Issue: []OperationOutcomeIssueComponent{
			OperationOutcomeIssueComponent{
				Severity:    severity,
				Code:        code,
				Diagnostics: getOpOutcomeDiagnostics(),
			},
		},
	}

	if detailsCode != "" {
		outcome.Issue[0].Details = &CodeableConcept{
			Coding: []Coding{
				Coding{
					Code:    detailsCode,
					System:  "http://hl7.org/fhir/ValueSet/operation-outcome",
					Display: detailsDisplay},
			},
			Text: detailsDisplay,
		}
	}

	if detailsCode == "" && detailsDisplay != "" {
		outcome.Issue[0].Details = &CodeableConcept{
			Coding: []Coding{
				Coding{
					Display: detailsDisplay},
			},
			Text: detailsDisplay,
		}
	}

	return outcome
}