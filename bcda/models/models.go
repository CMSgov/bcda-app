package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"
)

const (
	JobStatusPending          JobStatus = "Pending"
	JobStatusInProgress       JobStatus = "In Progress"
	JobStatusCompleted        JobStatus = "Completed"
	JobStatusArchived         JobStatus = "Archived"
	JobStatusExpired          JobStatus = "Expired"
	JobStatusFailed           JobStatus = "Failed"
	JobStatusCancelled        JobStatus = "Cancelled"
	JobStatusFailedExpired    JobStatus = "FailedExpired"    // JobStatusFailedExpired represents a job that failed whose data has been cleaned up
	JobStatusCancelledExpired JobStatus = "CancelledExpired" // JobStatusCancelledExpired represents a job that has been cancelled whose data has been cleaned up
)

var AllJobStatuses []JobStatus = []JobStatus{JobStatusPending, JobStatusInProgress, JobStatusCompleted,
	JobStatusArchived, JobStatusExpired, JobStatusFailed, JobStatusCancelled, JobStatusFailedExpired, JobStatusCancelledExpired}

type JobStatus string
type Job struct {
	ID              uint
	ACOID           uuid.UUID `json:"aco_id"`
	RequestURL      string    `json:"request_url"` // request_url
	Status          JobStatus `json:"status"`      // status
	TransactionTime time.Time // most recent data load transaction time from BFD
	JobCount        int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (j *Job) StatusMessage(numCompletedJobKeys int) string {
	if j.Status == JobStatusInProgress && j.JobCount > 0 {
		pct := float64(numCompletedJobKeys) / float64(j.JobCount) * 100
		return fmt.Sprintf("%s (%d%%)", j.Status, int(pct))
	}

	return string(j.Status)
}

// BlankFileName contains the naming convention for empty ndjson file
const BlankFileName string = "blank.ndjson"

type JobKey struct {
	ID    uint
	JobID uint `json:"job_id"`
	// Although que_job records are temporary, we store the ID to ensure
	// that workers are never duplicating job keys.
	QueJobID     *int64
	FileName     string
	ResourceType string
}

func (j *JobKey) IsError() bool {
	return strings.Contains(j.FileName, "-error.ndjson")
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
	TerminationDetails *Termination `json:"termination"`
}

// Denylisted returns bool based on TerminationDetails.
func (aco *ACO) Denylisted() bool {
	if aco.TerminationDetails != nil {
		if aco.TerminationDetails.DenylistType == Involuntary || aco.TerminationDetails.DenylistType == Voluntary {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
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
	// CreatedAt is automatically set by the database. This is only
	// set/pulled when querying data in the DB.
	CreatedAt time.Time
}

// "The MBI has 11 characters, like the Health Insurance Claim Number (HICN), which can have up to 11."
// https://www.cms.gov/Medicare/New-Medicare-Card/Understanding-the-MBI-with-Format.pdf
type CCLFBeneficiary struct {
	ID           uint
	FileID       uint
	MBI          string
	BlueButtonID string
}
