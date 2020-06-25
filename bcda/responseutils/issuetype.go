package responseutils

// Definition: A code that describes the type of issue.
// This value set includes codes from the following code systems:
// See: http://hl7.org/fhir/issue-type
const (
	Invalid          = "invalid"
	Structure        = "structure"
	Required         = "required"
	Value            = "value"
	Invariant        = "invariant"
	Security         = "security"
	Login            = "login"
	Unknown          = "unknown"
	Expired          = "expired"
	Forbidden        = "forbidden"
	Suppressed       = "suppressed"
	Processing       = "processing"
	Not_supported    = "not-supported"
	Duplicate        = "duplicate"
	Multiple_matches = "multiple-matches"
	Not_found        = "not-found"
	Deleted          = "deleted"
	Too_long         = "too-long"
	Code_invalid     = "code-invalid"
	Extension        = "extension"
	Too_costly       = "too-costly"
	Business_rule    = "business-rule"
	Conflict         = "conflict"
	Incomplete       = "incomplete"
	Transient        = "transient"
	Lock_error       = "lock-error"
	No_store         = "no-store"
	Exception        = "exception"
	Timeout          = "timeout"
	Throttled        = "throttled"
	Informational    = "informational"
)
