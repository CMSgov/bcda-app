package worker_types

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
)

const PrepareSharedJobKind = "PrepareSharedJob"

type PrepareSharedJobArgs struct {
	Job models.Job
	// ACOID                  uuid.UUID
	// CMSID                  string
	PartnerID string
	// CCLFFileNewID          uint
	// CCLFFileOldID          uint
	BFDPath string // eg const BFDV3Path = "/v3/fhir"
	// Version string
	// RequestType            constants.DataRequestType
	// ComplexDataRequestType string
	ResourceTypes []string
	Since         time.Time
	// TypeFilter             fhir.TypeFilterParameter
	CreationTime time.Time
	// ClaimsDate             time.Time
	// OptOutDate             time.Time
	TransactionID string
	MBIs          []string
	DataTypes     []string // "adjudicated", "partially-adjudicated"
}

func (args PrepareSharedJobArgs) Kind() string {
	return PrepareSharedJobKind
}
