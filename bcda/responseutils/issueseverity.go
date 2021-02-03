package responseutils

import (
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
)

// Definition: How the issue affects the success of the action.
// This value set includes codes from the following code systems:
// Defining URL: http://hl7.org/fhir/issue-severity
const (
	Fatal       = "fatal"
	Error       = "error"
	Warning     = "warning"
	Information = "information"
)

// issueSeverity returns the associated code value based on the supplied severity string
func issueSeverity(severity string) fhircodes.IssueSeverityCode_Value {
	switch severity {
	case Fatal:
		return fhircodes.IssueSeverityCode_FATAL
	case Error:
		return fhircodes.IssueSeverityCode_ERROR
	case Warning:
		return fhircodes.IssueSeverityCode_WARNING
	case Information:
		return fhircodes.IssueSeverityCode_INFORMATION
	default:
		return fhircodes.IssueSeverityCode_INVALID_UNINITIALIZED
	}
}
