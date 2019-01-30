package models

import (
	fhirmodels "github.com/eug48/fhir/models"
)

// BulkRequestResponse is the return from a request to initiate a bulk data collection process
// swagger:response BulkRequestResponse
type BulkRequestResponse struct {
	// The location where the job status can be checked
	ContentLocation string `json:"Content-Location"`
}

type operationOutcomeResponse struct {
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

// The requested path was not found. The body will contain a FHIR OperationOutcome resource in JSON format. https://www.hl7.org/fhir/operationoutcome.html
// swagger:response notFoundResponse
type notFoundResponse struct {
	// in: body
	Body operationOutcomeResponse
}

// There was a problem with the request. The body will contain a FHIR OperationOutcome resource in JSON format. https://www.hl7.org/fhir/operationoutcome.html Please refer to the body of the response for details.
// swagger:response badRequestResponse
type badRequestResponse struct {
	// in: body
	Body operationOutcomeResponse
}

// The requested resource is no longer available. The body will contain a FHIR OperationOutcome resource in JSON format. https://www.hl7.org/fhir/operationoutcome.html
// swagger:response goneResponse
type goneResponse struct {
	// in: body
	Body operationOutcomeResponse
}

// An error occurred. The body will contain a FHIR OperationOutcome resource in JSON format. https://www.hl7.org/fhir/operationoutcome.html Please refer to the body of the response for details.
// swagger:response errorResponse
type errorResponse struct {
	// in: body
	Body operationOutcomeResponse
}

// Data export job is in progress.
// swagger:response jobStatusResponse
type jobStatusResponse struct {
	// The status of the job progress
	XProgress string `json:"X-Progress"`
}

// JSON object containing a version field
// swagger:response VersionResponse
type VersionResponse struct {
	// in: body
	Body struct {
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

// File of newline-delimited JSON FHIR ExplanationOfBenefit objects
// swagger:response ExplanationOfBenefitNDJSON
type ExplanationOfBenefitNDJSON struct {
	// in: body
	// minimum items: 1
	Body *[]fhirmodels.ExplanationOfBenefit
}

// A JobStatus parameter model.
//
// This is used for operations that want the ID of a job in the path
// swagger:parameters jobStatus serveData
type JobIDParam struct {
	// ID of data export job
	//
	// in: path
	// required: true
	JobID int `json:"jobId"`
}

// swagger:parameters serveData
type FileParam struct {
	// Name of file to be downloaded
	// in: path
	// required: true
	Filename string `json:"filename"`
}

// swagger:parameters bulkPatientRequest bulkEOBRequest
type BulkRequestHeaders struct {
	// required: true
	// in: header
	// enum: respond-async
	Prefer string
}
