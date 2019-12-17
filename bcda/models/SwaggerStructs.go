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

type OperationOutcomeResponse struct {
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
type NotFoundResponse struct {
	// in: body
	Body OperationOutcomeResponse
}

// There was a problem with the request. The body will contain a FHIR OperationOutcome resource in JSON format. https://www.hl7.org/fhir/operationoutcome.html Please refer to the body of the response for details.
// swagger:response badRequestResponse
type BadRequestResponse struct {
	// in: body
	Body OperationOutcomeResponse
}

// The requested resource is no longer available. The body will contain a FHIR OperationOutcome resource in JSON format. https://www.hl7.org/fhir/operationoutcome.html
// swagger:response goneResponse
type GoneResponse struct {
	// in: body
	Body OperationOutcomeResponse
}

// An error occurred. The body will contain a FHIR OperationOutcome resource in JSON format. https://www.hl7.org/fhir/operationoutcome.html Please refer to the body of the response for details.
// swagger:response errorResponse
type ErrorResponse struct {
	// in: body
	Body OperationOutcomeResponse
}

// A bulk export job of this resource type is already in progress for the ACO.
// swagger:response tooManyRequestsResponse
type TooManyRequestsResponse struct {
}

// Data export job is in progress.
// swagger:response jobStatusResponse
type JobStatusResponse struct {
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

// JSON object containing an auth_provider field
// swagger:response AuthResponse
type AuthResponse struct {
	// in: body
	Body struct {
		// Required: true
		Version string `json:"auth_provider"`
	}
}

// FHIR CapabilityStatement in JSON format
// swagger:response MetadataResponse
type MetadataResponse struct {
	// in: body
	Body fhirmodels.CapabilityStatement `json:"body,omitempty"`
}

// File of newline-delimited JSON FHIR objects
// swagger:response FileNDJSON
type FileNDJSON struct {
	// in: body
	Body string `json:"ndjson"`
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

// swagger:parameters bulkPatientRequest bulkGroupRequest
type ResourceTypeParam struct {
	// Resource types requested
	// in: query
	// style: form
	// explode: false
	ResourceType []string `json:"_type"`
}

// swagger:parameters bulkPatientRequest bulkGroupRequest
type BulkRequestHeaders struct {
	// required: true
	// in: header
	// enum: respond-async
	Prefer string
}

// A BulkGroupRequest parameter model.
//
// This is used for operations that want the groupID of a group in the path
// swagger:parameters bulkGroupRequest
type GroupIDParam struct {
	// ID of group export
	// in: path
	// required: true
	GroupID string `json:"groupId"`
}

// JSON with a valid JWT
// swagger:response tokenResponse
type TokenResponse struct {
	// in: body
	Body struct {
		// Required: true
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
}

// Missing credentials
// swagger:response missingCredentials
type MissingCredentials struct{}

// Unauthorized. The provided credentials are invalid for the requested resource.
// swagger:response invalidCredentials
type InvalidCredentials struct{}

// Server error
// swagger:response serverError
type ServerError struct{}
