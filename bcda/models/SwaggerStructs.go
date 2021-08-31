package models

import (
	"time"

	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
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

// The job has been deleted.
// swagger:response deleteJobResponse
type DeleteJobResponse struct {
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

// JSON object containing a cclf_files field
// swagger:response AttributionFileStatusResponse
type AttributionFileStatusResponse struct {
	// in: body
	Body AttributionFilesParam `json:"body,omitempty"`
}

type AttributionFilesParam struct {
	IngestionDates []IngestionDatesStatusParam `json:"ingestion_dates"`
}

type IngestionDatesStatusParam struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
}

// File of newline-delimited JSON FHIR objects
// swagger:response FileNDJSON
type FileNDJSON struct {
	// Header defining encoding type used
	// enum: gzip
	ContentEncoding string `json:"Content-Encoding"`
	// in: body
	Body NDJSON
}

// swagger:model
type NDJSON string

// A JobStatus parameter model.
//
// This is used for operations that want the ID of a job in the path
// swagger:parameters jobStatus jobStatusV2 serveData deleteJob deleteJobV2
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

// swagger:parameters serveData
type ServeDataRequestHeaders struct {
	// Encoding type to use
	// in: header
	// enum: gzip
	AcceptEncoding string `json:"Accept-Encoding"`
}

// swagger:parameters bulkPatientRequest bulkGroupRequest
type ResourceTypeParam struct {
	// Resource types requested
	// in: query
	// style: form
	// explode: false
	ResourceType []string `json:"_type"`
}

// swagger:parameters bulkPatientRequestV2 bulkGroupRequestV2
type ResourceTypeParamV2 struct {
	// Resource types requested
	// in: query
	// style: form
	// explode: false
	// required: true
	// items.enum: Coverage,Patient,ExplanationOfBenefit
	ResourceType []string `json:"_type"`
}

// swagger:parameters bulkPatientRequest bulkGroupRequest bulkPatientRequestV2 bulkGroupRequestV2
type SinceParam struct {
	// Only include resource versions that were created at or after the given instant in time.  Format of string must align with the FHIR Instant datatype (i.e., `2020-02-13T08:00:00.000-05:00`)
	// in: query
	// required: false
	DateTime string `json:"_since"`
}

// swagger:parameters bulkPatientRequest bulkGroupRequest bulkPatientRequestV2 bulkGroupRequestV2
type BulkRequestHeaders struct {
	// required: true
	// in: header
	// enum: respond-async
	Prefer string
}

// A BulkGroupRequest parameter model.
//
// This is used for operations that want the groupID of a group in the path
// swagger:parameters bulkGroupRequest bulkGroupRequestV2
type GroupIDParam struct {
	// ID of group export
	// in: path
	// required: true
	// enum: all,runout
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

// Welcome message
// swagger:response welcome
type Welcome struct{}
