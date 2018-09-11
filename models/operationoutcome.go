package models

type OperationOutcome struct {
	Issue OperationOutcomeIssue `json:"issue"`
}

type OperationOutcomeIssue struct {
	Severity    string           `json:"severity"`
	Code        string           `json:"code"`
	Details     *CodeableConcept `json:"details"`
	Diagnostics string           `json:"diagnostics"`
	Location    []string         `json:"location"`
	Expression  []string         `json:"expression"`
}

type CodeableConcept struct {
	Coding *Coding `json:"coding"`
	Text   string  `json:"text"`
}

type Coding struct {
	System       string `json:"system"`
	Version      string `json:"version"`
	Code         string `json:"code"`
	Display      string `json:"display"`
	UserSelected bool   `json:"userSelected"`
}
