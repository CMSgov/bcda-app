package models

import fhirmodels "github.com/eug48/fhir/models"

// BulkRequestResponse is the return from a request to initiate a bulk data collection process
// swagger:response BulkRequestResponse
type BulkRequestResponse struct {
	// The location where the job status can be checked
	ContentLocation string `json:"Content-Location"`
}

// Operation Outcome follows HL7 FHIR Spec https://www.hl7.org/fhir/operationoutcome.html
// swagger:response FHIRResponse
type OperationOutcome struct {
	// A Valid FHIR Response
	// in:body
	FHIRResponse struct {
		// OperationOutcome
		ResourceType string
		// A single issue associated with the action

		Issue struct {
			// Severity of the outcome: fatal | error | warning | information
			// Required: true
			Severity string
			//Error or warning code
			// Required: true
			Code string
			// Additional details about the error
			// Required: true
			Details string
			// Additional diagnostic information about the issue
			// Required: true
			Diagnostics string
			// Path of element(s) related to issue
			// Required: true
			Location string
			// FHIRPath of element(s) related to issue
			// Required: true
			Expression string
		}
	}
}

// A generic HTTP error Model.  Should only be used for well documented error types such as 404
// swagger:response ErrorModel
type ErrorModel struct {
	// Message contains additional information about this error
	Message string
}

// JobStatus defines the status of a specific Job defined by the ID
// swagger:response JobStatus
type JobStatus struct {
	// The status of the job progress
	XProgress string `json:"X-Progress"`
}

// JSON object containing a version field
// swagger:response VersionResponse
type VersionResponse struct {
	// Version
	// in: body
	Body struct {
		// Version number
		// Required: true
		Version string `json:"version"`
	}
}

// FHIR CapabilityStatement in JSON format
// swagger:response MetadataResponse
type MetadataResponse struct {
	// in: body
	Body fhirmodels.CapabilityStatement `json:"body,omitempty"`
}

// A JobStatus parameter model.
//
// This is used for operations that want the ID of a job in the path
// swagger:parameters jobStatus
type JobStatusParam struct {
	// The ID of the job
	//
	// in: path
	// required: true
	JobID int `json:"jobid"`
}
