package worker_types

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/api/request_types"
	"github.com/CMSgov/bcda-app/bcda/models"

	"github.com/pborman/uuid"
)

const PrepareSharedJobKind = "PrepareSharedJob"

type PrepareSharedJobArgs struct {
	ACOID         uuid.UUID
	BFDPath       string // eg const BFDV3Path = "/v3/fhir"
	CMSID         string
	CreationTime  time.Time
	Job           models.Job
	Queries       []request_types.SharedExportQueryData
	TransactionID string
}

func (args PrepareSharedJobArgs) Kind() string {
	return PrepareSharedJobKind
}
