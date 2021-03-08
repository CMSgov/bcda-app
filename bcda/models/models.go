package models

import (
	"fmt"
	"time"

	"github.com/pborman/uuid"
)

const (
	JobStatusPending       JobStatus = "Pending"
	JobStatusInProgress    JobStatus = "In Progress"
	JobStatusCompleted     JobStatus = "Completed"
	JobStatusArchived      JobStatus = "Archived"
	JobStatusExpired       JobStatus = "Expired"
	JobStatusFailed        JobStatus = "Failed"
	JobStatusCancelled     JobStatus = "Cancelled"
	JobStatusFailedExpired JobStatus = "FailedExpired" // JobStatusFailedExpired represents on job that failed whose data has been cleaned up
)

var AllJobStatuses []JobStatus = []JobStatus{JobStatusPending, JobStatusInProgress, JobStatusCompleted,
	JobStatusArchived, JobStatusExpired, JobStatusFailed, JobStatusCancelled, JobStatusFailedExpired}

type JobStatus string
type Job struct {
	ID                uint
	ACOID             uuid.UUID `json:"aco_id"`
	RequestURL        string    `json:"request_url"` // request_url
	Status            JobStatus `json:"status"`      // status
	TransactionTime   time.Time // most recent data load transaction time from BFD
	JobCount          int
	CompletedJobCount int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (j *Job) StatusMessage() string {
	if j.Status == JobStatusInProgress && j.JobCount > 0 {
		pct := float64(j.CompletedJobCount) / float64(j.JobCount) * 100
		return fmt.Sprintf("%s (%d%%)", j.Status, int(pct))
	}

	return string(j.Status)
}

// BlankFileName contains the naming convention for empty ndjson file
const BlankFileName string = "blank.ndjson"

type JobKey struct {
	ID           uint
	JobID        uint `json:"job_id"`
	FileName     string
	ResourceType string
}

// ACO represents an Accountable Care Organization.
type ACO struct {
	ID                 uint
	UUID               uuid.UUID    `json:"uuid"`
	CMSID              *string      `json:"cms_id"`
	Name               string       `json:"name"`
	ClientID           string       `json:"client_id"`
	GroupID            string       `json:"group_id"`
	SystemID           string       `json:"system_id"`
	AlphaSecret        string       `json:"alpha_secret"`
	PublicKey          string       `json:"public_key"`
	Blacklisted        bool         `json:"blacklisted"`
	TerminationDetails *Termination `json:"termination"`
}

type CCLFFileType int16

const (
	FileTypeDefault CCLFFileType = iota
	FileTypeRunout
)

// String returns the letter associated with the CCLFFileType.
func (t CCLFFileType) String() string {
	return [...]string{"Y", "R"}[t]
}

type CCLFFile struct {
	ID              uint
	CCLFNum         int
	Name            string
	ACOCMSID        string
	Timestamp       time.Time
	PerformanceYear int
	ImportStatus    string
	Type            CCLFFileType
}

// "The MBI has 11 characters, like the Health Insurance Claim Number (HICN), which can have up to 11."
// https://www.cms.gov/Medicare/New-Medicare-Card/Understanding-the-MBI-with-Format.pdf
type CCLFBeneficiary struct {
	ID           uint
	FileID       uint
	MBI          string
	BlueButtonID string
}

type SuppressionFile struct {
	ID           uint
	Name         string
	Timestamp    time.Time
	ImportStatus string
}

type Suppression struct {
	ID                  uint
	FileID              uint
	MBI                 string
	SourceCode          string
	EffectiveDt         time.Time
	PrefIndicator       string
	SAMHSASourceCode    string
	SAMHSAEffectiveDt   time.Time
	SAMHSAPrefIndicator string
	ACOCMSID            string
	BeneficiaryLinkKey  int
}

type JobEnqueueArgs struct {
	ID              int
	ACOID           string
	BeneficiaryIDs  []string
	ResourceType    string
	Since           string
	TransactionTime time.Time
	ServiceDate     time.Time
	BBBasePath      string
}
