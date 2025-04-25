package worker_types

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/pborman/uuid"
)

type PrepareJobArgs struct {
	Job                    models.Job
	ACOID                  uuid.UUID
	CMSID                  string
	CCLFFileNewID          uint
	CCLFFileOldID          uint
	BFDPath                string
	RequestType            constants.DataRequestType
	ComplexDataRequestType string
	ResourceTypes          []string
	Since                  time.Time
	CreationTime           time.Time
	ClaimsDate             time.Time
	OptOutDate             time.Time
	TransactionID          string
}

func (args PrepareJobArgs) Kind() string {
	return constants.PrepareJobKind
}
