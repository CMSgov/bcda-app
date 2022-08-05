package responseutils

// Internal codes: These will be modified over time
const (
	TokenErr        = "Invalid Token"
	DbErr           = "Database Error"
	FormatErr       = "Formatting Error"
	BbErr           = "Blue Button Error"
	InternalErr     = "Internal Error"
	RequestErr      = "Request Error"
	UnauthorizedErr = "Unauthorized Error"
	NotFoundErr     = "Not Found Error"
	DeletedErr      = "Deleted Error"
)

//External messaging: Messages that will be in response body
const (
	UnknownEntityErr = "unknown entity"
)

const (
	JobFailed       = "Job Failed"
	DetailJobFailed = "Service encountered numerous errors. Job failed to complete."
)
