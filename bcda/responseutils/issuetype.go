package responseutils

// Definition: A code that describes the type of issue.
// This value set includes codes from the following code systems:
// See: http://hl7.org/fhir/issue-type
const (
	Invalid       = "Invalid Content"
	Structure     = "Structural Issue"
	Required      = "Required element missing"
	Value         = "Element value invalid"
	Invariant     = "Validation rule failed"
	Security      = "Security Problem"
	Login         = "Login Required"
	Unknown       = "Unknown User"
	Expired       = "Session Expired"
	Forbidden     = "Forbidden"
	Suppressed    = "Information Suppressed"
	Processing    = "Processing Failure"
	Not_supported = "Content not supported"
	Duplicate     = "Duplicate"
	Not_found     = "Not Found"
	Too_long      = "Content Too Long"
	Code_invalid  = "Invalid Code"
	Extension     = "Unacceptable Extension"
	Too_costly    = "Operation Too Costly"
	Business_rule = "Business Rule Violation"
	Conflict      = "Edit Version Conflict"
	Incomplete    = "Incomplete Results"
	Transient     = "Transient Issue"
	Lock_error    = "Lock Error"
	No_store      = "No Store Available"
	Exception     = "Exception"
	Timeout       = "Timeout"
	Throttled     = "Throttled"
	Informational = "Informational Note"
)
