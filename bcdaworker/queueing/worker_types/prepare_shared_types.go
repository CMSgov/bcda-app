package worker_types

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/api/request_types"
	"github.com/CMSgov/bcda-app/bcda/models"

	"github.com/pborman/uuid"
)

const PrepareSharedJobKind = "PrepareSharedJob"

// TODO: (BCDA-10574)
type PrepareSharedJobArgs struct {
	ACOID         uuid.UUID
	BFDPath       string // eg const BFDV3Path = "/v3/fhir"
	CMSID         string
	CreationTime  time.Time
	Job           models.Job
	Queries       []request_types.SharedExportQuery
	TransactionID string
	// PartnerID string
	// CCLFFileNewID          uint
	// CCLFFileOldID          uint
	// Version string
	// RequestType            constants.DataRequestType
	// ComplexDataRequestType string
	// ResourceTypes []string
	// Since         time.Time
	// TypeFilter             fhir.TypeFilterParameter
	// ClaimsDate             time.Time
	// OptOutDate             time.Time
	// MBIs          []string
	// DataTypes     []string // "adjudicated", "partially-adjudicated"

}

func (args PrepareSharedJobArgs) Kind() string {
	return PrepareSharedJobKind
}
