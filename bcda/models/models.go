package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
)

const (
	JobStatusPending    JobStatus = "Pending"
	JobStatusInProgress JobStatus = "In Progress"
	JobStatusCompleted  JobStatus = "Completed"
	JobStatusArchived   JobStatus = "Archived"
	JobStatusExpired    JobStatus = "Expired"
	JobStatusFailed     JobStatus = "Failed"
	JobStatusCancelled  JobStatus = "Cancelled"
)

var AllJobStatuses []JobStatus = []JobStatus{JobStatusPending, JobStatusInProgress,
	JobStatusCompleted, JobStatusArchived, JobStatusExpired, JobStatusFailed, JobStatusCancelled}

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
	ID          uint
	UUID        uuid.UUID `json:"uuid"`
	CMSID       *string   `json:"cms_id"`
	Name        string    `json:"name"`
	ClientID    string    `json:"client_id"`
	GroupID     string    `json:"group_id"`
	SystemID    string    `json:"system_id"`
	AlphaSecret string    `json:"alpha_secret"`
	PublicKey   string    `json:"public_key"`
	Blacklisted bool      `json:"blacklisted"`
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

// This method will ensure that a valid BlueButton ID is returned.
// If you use cclfBeneficiary.BlueButtonID you will not be guaranteed a valid value
func (cclfBeneficiary *CCLFBeneficiary) GetBlueButtonID(bb client.APIClient) (blueButtonID string, err error) {
	modelIdentifier := cclfBeneficiary.MBI

	blueButtonID, err = GetBlueButtonID(bb, modelIdentifier, "beneficiary", cclfBeneficiary.ID)
	if err != nil {
		return "", err
	}
	return blueButtonID, nil
}

func GetBlueButtonID(bb client.APIClient, modelIdentifier, reqType string, modelID uint) (blueButtonID string, err error) {
	hashedIdentifier := client.HashIdentifier(modelIdentifier)

	jsonData, err := bb.GetPatientByIdentifierHash(hashedIdentifier)
	if err != nil {
		return "", err
	}
	var patient Patient
	err = json.Unmarshal([]byte(jsonData), &patient)
	if err != nil {
		log.Error(err)
		return "", err
	}

	if len(patient.Entry) == 0 {

		err = fmt.Errorf("patient identifier not found at Blue Button for CCLF %s ID: %v", reqType, modelID)

		log.Error(err)
		return "", err
	}
	var foundIdentifier = false
	var foundBlueButtonID = false
	blueButtonID = patient.Entry[0].Resource.ID
	for _, identifier := range patient.Entry[0].Resource.Identifier {
		if strings.Contains(identifier.System, "us-mbi") {
			if identifier.Value == modelIdentifier {
				foundIdentifier = true
			}
		} else if strings.Contains(identifier.System, "bene_id") && identifier.Value == blueButtonID {
			foundBlueButtonID = true
		}
	}
	if !foundIdentifier {
		err = fmt.Errorf("Identifier not found")
		log.Error(err)
		return "", err
	}
	if !foundBlueButtonID {
		err = fmt.Errorf("Blue Button identifier not found in the identifiers")
		log.Error(err)
		return "", err
	}

	return blueButtonID, nil
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
