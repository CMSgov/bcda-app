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
	ACOID            string
	BBBasePath       string
	BeneficiaryIDs   []string
	ClaimsWindow     ClaimsWindow
	CMSID            string
	DataType         string
	ID               int // parent Job ID
	RequestingSystem string
	ResourceType     string
	Since            string
	TransactionID    string
	TransactionTime  time.Time
	TypeFilter       fhir.TypeFilterSubquery
}

// Needed by River (queue library)
func (jobargs JobEnqueueArgs) Kind() string {
	return QUE_PROCESS_JOB
}
