package request_types

import (
	"encoding/json"
	"time"

	"github.com/CMSgov/bcda-app/bcda/fhir"
)

// used by Patient and Group $export GET requests
type RequestParameters struct {
	Since         time.Time
	ResourceTypes []string
	Version       string // e.g. v1, v2
	RequestURL    string
	TypeFilter    fhir.TypeFilterSubquery
}

// used by Shared Export POST requests
// TODO: (BCDA-10505) add body validation
type SharedExportPayload struct {
	Queries     []SharedExportQuery `json:"queries"`
	SourceQuery string              `json:"source_query,omitempty"`
	System      string              `json:"system"`
	TrackingID  string              `json:"tracking_id,omitempty"`
	Version     string              `json:"version,omitempty"`
}

type SharedExportQuery struct {
	FormData      []SharedExportFormData `json:"form_data,omitempty"`
	Headers       map[string]string      `json:"headers,omitempty"`
	MBIs          []string               `json:"mbis"`
	ResourceTypes []string               `json:"resource_types"`
}

type SharedExportFormData struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

// derived struct for internal use
type SharedExportQueryData struct {
	SharedExportQuery
	Since       time.Time
	SourceQuery string
	System      string
	TrackingID  string
	TypeFilter  fhir.TypeFilterSubquery
	Version     string
}
