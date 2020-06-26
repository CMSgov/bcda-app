package responseutils

// Definition: How the issue affects the success of the action.
// This value set includes codes from the following code systems:
// Defining URL: http://hl7.org/fhir/issue-severity
const (
	Fatal       = "fatal"
	Error       = "error"
	Warning     = "warning"
	Information = "information"
)
