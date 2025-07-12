package worker_types

import "time"

const QUE_PROCESS_JOB = "ProcessJob"

type JobEnqueueArgs struct {
	ID              int // parent Job ID
	ACOID           string
	CMSID           string
	BeneficiaryIDs  []string
	ResourceType    string
	Since           string
	TypeFilter      [][]string
	TransactionID   string
	TransactionTime time.Time
	BBBasePath      string
	ClaimsWindow    struct {
		LowerBound time.Time
		UpperBound time.Time
	}
	DataType string
}

// Needed by River (queue library)
func (jobargs JobEnqueueArgs) Kind() string {
	return QUE_PROCESS_JOB
}
