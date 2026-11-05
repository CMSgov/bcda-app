package request_types

import (
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
	Queries []SharedExportQuery `json:"queries"`
	System  string              `json:"system"`
}

type SharedExportQuery struct {
	DataTypes     []string           `json:"data_types"`
	MBIs          []string           `json:"mbis"`
	ResourceTypes []string           `json:"resource_types"`
	SourceQuery   string             `json:"source_query"`
	ParsedQuery   SharedExportParams `json:"parsed_query"`
	Version       string             `json:"version"`
}

type SharedExportParams struct {
	Headers    map[string]string      `json:"headers,omitempty"`
	Parameters SharedExportParameters `json:"parameters,omitempty"`
}

type SharedExportParameters struct {
	Since      string                  `json:"_since,omitempty"`
	TypeFilter fhir.TypeFilterSubquery `json:"_typeFilter,omitempty"`
}
