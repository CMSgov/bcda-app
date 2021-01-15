package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	JobStatusPending    JobStatus = "Pending"
	JobStatusInProgress JobStatus = "In Progress"
	JobStatusCompleted  JobStatus = "Completed"
	JobStatusArchived   JobStatus = "Archived"
	JobStatusExpired    JobStatus = "Expired"
	JobStatusFailed     JobStatus = "Failed"
)

var AllJobStatuses []JobStatus = []JobStatus{JobStatusPending, JobStatusInProgress,
	JobStatusCompleted, JobStatusArchived, JobStatusExpired, JobStatusFailed}

type JobStatus string
type Job struct {
	ID                uint
	ACO               ACO       `gorm:"foreignkey:ACOID;association_foreignkey:UUID"` // aco
	ACOID             uuid.UUID `gorm:"type:char(36)" json:"aco_id"`
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
	gorm.Model
	JobID        uint   `gorm:"primary_key" json:"job_id"`
	FileName     string `gorm:"type:char(127)"`
	ResourceType string
}

// ACO represents an Accountable Care Organization.
type ACO struct {
	gorm.Model
	UUID        uuid.UUID `gorm:"primary_key;type:char(36)" json:"uuid"`
	CMSID       *string   `gorm:"type:varchar(5);unique" json:"cms_id"`
	Name        string    `json:"name"`
	ClientID    string    `json:"client_id"`
	GroupID     string    `json:"group_id"`
	SystemID    string    `json:"system_id"`
	AlphaSecret string    `json:"alpha_secret"`
	PublicKey   string    `json:"public_key"`
	Blacklisted bool      `json:"blacklisted"`
}

type CCLFBeneficiaryXref struct {
	gorm.Model
	FileID        uint   `gorm:"not null"`
	XrefIndicator string `json:"xref_indicator"`
	CurrentNum    string `json:"current_number"`
	PrevNum       string `json:"previous_number"`
	PrevsEfctDt   string `json:"effective_date"`
	PrevsObsltDt  string `json:"obsolete_date"`
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
	CCLFNum         int          `gorm:"not null"`
	Name            string       `gorm:"not null;UNIQUE_INDEX:idx_cclf_files_name_aco_cms_id_key"`
	ACOCMSID        string       `gorm:"column:aco_cms_id;UNIQUE_INDEX:idx_cclf_files_name_aco_cms_id_key"`
	Timestamp       time.Time    `gorm:"not null"`
	PerformanceYear int          `gorm:"not null"`
	ImportStatus    string       `gorm:"column:import_status"`
	Type            CCLFFileType `gorm:"column:type;default:0"`
}

// "The MBI has 11 characters, like the Health Insurance Claim Number (HICN), which can have up to 11."
// https://www.cms.gov/Medicare/New-Medicare-Card/Understanding-the-MBI-with-Format.pdf
type CCLFBeneficiary struct {
	ID           uint
	CCLFFile     CCLFFile `gorm:"foreignkey:file_id;association_foreignkey:id"`
	FileID       uint     `gorm:"not null;index:idx_cclf_beneficiaries_file_id"`
	MBI          string   `gorm:"type:char(11);not null;index:idx_cclf_beneficiaries_mbi"`
	BlueButtonID string   `gorm:"type: text;index:idx_cclf_beneficiaries_bb_id"`
}

type SuppressionFile struct {
	gorm.Model
	Name         string    `gorm:"not null;unique"`
	Timestamp    time.Time `gorm:"not null"`
	ImportStatus string    `gorm:"column:import_status"`
}

type Suppression struct {
	gorm.Model
	FileID              uint      `gorm:"not null"`
	MBI                 string    `gorm:"type:varchar(11);index:idx_suppression_mbi"`
	SourceCode          string    `gorm:"type:varchar(5)"`
	EffectiveDt         time.Time `gorm:"column:effective_date"`
	PrefIndicator       string    `gorm:"column:preference_indicator;type:char(1)"`
	SAMHSASourceCode    string    `gorm:"type:varchar(5)"`
	SAMHSAEffectiveDt   time.Time `gorm:"column:samhsa_effective_date"`
	SAMHSAPrefIndicator string    `gorm:"column:samhsa_preference_indicator;type:char(1)"`
	ACOCMSID            string    `gorm:"column:aco_cms_id;type:char(5)"`
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

// This is not a persistent model so it is not necessary to include in GORM auto migrate.
// swagger:ignore
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
