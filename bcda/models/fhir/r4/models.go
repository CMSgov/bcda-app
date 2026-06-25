package r4

type Bundle struct {
	ResourceType string        `json:"resourceType"`
	Type         string        `json:"type"`
	Total        uint32        `json:"total"`
	Entry        []BundleEntry `json:"entry,omitempty"`
}

type BundleEntry struct {
	Resource any `json:"resource,omitempty"`
}

type Task struct {
	ResourceType    string       `json:"resourceType"`
	Identifier      []Identifier `json:"identifier,omitempty"`
	Status          string       `json:"status"`
	Intent          string       `json:"intent"`
	Input           []Parameter  `json:"input,omitempty"`
	ExecutionPeriod Period       `json:"executionPeriod,omitempty"`
}

type Identifier struct {
	Use    string `json:"use,omitempty"`
	System string `json:"system,omitempty"`
	Value  string `json:"value,omitempty"`
}

type Parameter struct {
	Type  CodeableConcept `json:"type"`
	Value any             `json:"valueString,omitempty"` // Can be valueString or valueX. In JSON, choice types serialize as valueString
}

type CodeableConcept struct {
	Coding []Coding `json:"coding,omitempty"`
	Text   string   `json:"text,omitempty"`
}

type Coding struct {
	System  string `json:"system,omitempty"`
	Code    string `json:"code,omitempty"`
	Display string `json:"display,omitempty"`
}

type Period struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

type IssueTypeCode string

const (
	IssueTypeCodeException  IssueTypeCode = "exception"
	IssueTypeCodeNotFound   IssueTypeCode = "not-found"
	IssueTypeCodeStructure  IssueTypeCode = "structure"
	IssueTypeCodeProcessing IssueTypeCode = "processing"
	IssueTypeCodeForbidden  IssueTypeCode = "forbidden"
)

type IssueSeverityCode string

const (
	IssueSeverityFatal       IssueSeverityCode = "fatal"
	IssueSeverityError       IssueSeverityCode = "error"
	IssueSeverityWarning     IssueSeverityCode = "warning"
	IssueSeverityInformation IssueSeverityCode = "information"
)

type OperationOutcome struct {
	ResourceType string  `json:"resourceType"`
	Issue        []Issue `json:"issue"`
}

type Issue struct {
	Severity    IssueSeverityCode `json:"severity"`
	Code        IssueTypeCode     `json:"code"`
	Diagnostics string            `json:"diagnostics,omitempty"`
	Details     *CodeableConcept  `json:"details,omitempty"`
}

type CapabilityStatement struct {
	ResourceType   string                    `json:"resourceType"`
	Status         string                    `json:"status"`
	Date           string                    `json:"date"`
	Publisher      string                    `json:"publisher,omitempty"`
	Kind           string                    `json:"kind"`
	Instantiates   []string                  `json:"instantiates,omitempty"`
	Software       Software                  `json:"software,omitempty"`
	Implementation Implementation            `json:"implementation,omitempty"`
	FhirVersion    string                    `json:"fhirVersion"`
	Format         []string                  `json:"format"`
	Rest           []CapabilityStatementRest `json:"rest,omitempty"`
}

type Software struct {
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	ReleaseDate string `json:"releaseDate,omitempty"`
}

type Implementation struct {
	Description string `json:"description,omitempty"`
	Url         string `json:"url,omitempty"`
}

type CapabilityStatementRest struct {
	Mode        string          `json:"mode"`
	Security    *Security       `json:"security,omitempty"`
	Interaction []Interaction   `json:"interaction,omitempty"`
	Resource    []RestResource  `json:"resource,omitempty"`
	Operation   []RestOperation `json:"operation,omitempty"`
}

type RestResource struct {
	Type        string          `json:"type"`
	Operation   []RestOperation `json:"operation,omitempty"`
	SearchParam []SearchParam   `json:"searchParam,omitempty"`
}

type SearchParam struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Documentation string `json:"documentation,omitempty"`
}

type Security struct {
	Cors      bool              `json:"cors,omitempty"`
	Service   []CodeableConcept `json:"service,omitempty"`
	Extension []Extension       `json:"extension,omitempty"`
}

type Extension struct {
	Url       string      `json:"url"`
	ValueUri  string      `json:"valueUri,omitempty"`
	Extension []Extension `json:"extension,omitempty"`
}

type Interaction struct {
	Code string `json:"code"`
}

type RestOperation struct {
	Name          string `json:"name"`
	Definition    string `json:"definition"`
	Documentation string `json:"documentation,omitempty"`
}
