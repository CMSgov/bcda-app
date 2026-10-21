package worker_types

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/fhir"
)

const QUE_PROCESS_JOB = "ProcessJob"

type ClaimsWindow struct {
	Earliest time.Time `json:"Earliest,omitempty"`
	Latest   time.Time `json:"Latest,omitempty"`
}

type JobEnqueueArgs struct {
	ID              int // parent Job ID
	ACOID           string
	CMSID           string
	BeneficiaryIDs  []string
	ResourceType    string
	Since           string
	TypeFilter      fhir.TypeFilterSubquery
	TransactionID   string
	TransactionTime time.Time
	BBBasePath      string
	ClaimsWindow    ClaimsWindow
	DataType        string
}

// Needed by River (queue library)
func (jobargs JobEnqueueArgs) Kind() string {
	return QUE_PROCESS_JOB
}
