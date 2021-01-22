package models

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	authclient "github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/auth/rsautils"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
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
	JobStatusCancelled  JobStatus = "Cancelled"
)

type JobStatus string
type Job struct {
	gorm.Model
	ACO               ACO       `gorm:"foreignkey:ACOID;association_foreignkey:UUID"` // aco
	ACOID             uuid.UUID `gorm:"type:char(36)" json:"aco_id"`
	RequestURL        string    `json:"request_url"` // request_url
	Status            JobStatus `json:"status"`      // status
	TransactionTime   time.Time // most recent data load transaction time from BFD
	JobCount          int
	CompletedJobCount int
}

func (job *Job) CheckCompletedAndCleanup(db *gorm.DB) (bool, error) {

	// Trivial case, no need to keep going
	if job.Status == JobStatusCompleted {
		return true, nil
	}

	var completedJobs int64
	db.Model(&JobKey{}).Where("job_id = ?", job.ID).Count(&completedJobs)

	if int(completedJobs) >= job.JobCount {

		staging := fmt.Sprintf("%s/%d", os.Getenv("FHIR_STAGING_DIR"), job.ID)
		payload := fmt.Sprintf("%s/%d", os.Getenv("FHIR_PAYLOAD_DIR"), job.ID)

		files, err := ioutil.ReadDir(staging)
		if err != nil {
			log.Error(err)
		}

		for _, f := range files {
			oldPath := fmt.Sprintf("%s/%s", staging, f.Name())
			newPath := fmt.Sprintf("%s/%s", payload, f.Name())
			err := os.Rename(oldPath, newPath)
			if err != nil {
				log.Error(err)
			}
		}
		err = os.Remove(staging)
		if err != nil {
			log.Error(err)
		}
		return true, db.Model(&job).Update("status", JobStatusCompleted).Error
	}

	return false, nil
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

// GetPublicKey returns the ACO's public key.
func (aco *ACO) GetPublicKey() (*rsa.PublicKey, error) {
	var key string
	if strings.ToLower(os.Getenv("BCDA_AUTH_PROVIDER")) == "ssas" {
		ssas, err := authclient.NewSSASClient()
		if err != nil {
			return nil, errors.Wrap(err, "cannot retrieve public key for ACO "+aco.UUID.String())
		}

		systemID, err := strconv.Atoi(aco.ClientID)
		if err != nil {
			return nil, errors.Wrap(err, "cannot retrieve public key for ACO "+aco.UUID.String())
		}

		keyBytes, err := ssas.GetPublicKey(systemID)
		if err != nil {
			return nil, errors.Wrap(err, "cannot retrieve public key for ACO "+aco.UUID.String())
		}

		key = string(keyBytes)
	} else {
		key = aco.PublicKey
	}
	return rsautils.ReadPublicKey(key)
}

func (aco *ACO) SavePublicKey(publicKey io.Reader) error {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	k, err := ioutil.ReadAll(publicKey)
	if err != nil {
		return errors.Wrap(err, "cannot read public key for ACO "+aco.UUID.String())
	}

	key, err := rsautils.ReadPublicKey(string(k))
	if err != nil || key == nil {
		return errors.Wrap(err, "invalid public key for ACO "+aco.UUID.String())
	}

	aco.PublicKey = string(k)
	err = db.Save(&aco).Error
	if err != nil {
		return errors.Wrap(err, "cannot save public key for ACO "+aco.UUID.String())
	}

	return nil
}

// This exists to provide a known static keys used for ACO's in our alpha tests.
// This key is not meant to protect anything and both halves will be made available publicly
func GetATOPublicKey() *rsa.PublicKey {
	fmt.Println("Looking for a key at:")
	fmt.Println(os.Getenv("ATO_PUBLIC_KEY_FILE"))
	atoPublicKeyFile, err := os.Open(os.Getenv("ATO_PUBLIC_KEY_FILE"))
	if err != nil {
		fmt.Println("failed to open file")
		panic(err)
	}
	return utils.OpenPublicKeyFile(atoPublicKeyFile)
}

func GetATOPrivateKey() *rsa.PrivateKey {
	atoPrivateKeyFile, err := os.Open(os.Getenv("ATO_PRIVATE_KEY_FILE"))
	if err != nil {
		panic(err)
	}
	return utils.OpenPrivateKeyFile(atoPrivateKeyFile)
}

// CreateACO creates an ACO with the provided name and CMS ID.
func CreateACO(name string, cmsID *string) (uuid.UUID, error) {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	id := uuid.NewRandom()

	// TODO: remove ClientID below when a future refactor removes the need
	//    for every ACO to have a client_id at creation
	aco := ACO{Name: name, CMSID: cmsID, UUID: id, ClientID: id.String()}
	db.Create(&aco)

	return aco.UUID, db.Error
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
	gorm.Model
	CCLFNum         int          `gorm:"not null"`
	Name            string       `gorm:"not null;UNIQUE_INDEX:idx_cclf_files_name_aco_cms_id_key"`
	ACOCMSID        string       `gorm:"column:aco_cms_id;UNIQUE_INDEX:idx_cclf_files_name_aco_cms_id_key"`
	Timestamp       time.Time    `gorm:"not null"`
	PerformanceYear int          `gorm:"not null"`
	ImportStatus    string       `gorm:"column:import_status"`
	Type            CCLFFileType `gorm:"column:type;default:0"`
}

func (cclfFile *CCLFFile) Delete() error {
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	err := db.Unscoped().Where("file_id = ?", cclfFile.ID).Delete(&CCLFBeneficiary{}).Error
	if err != nil {
		return err
	}
	return db.Unscoped().Delete(&cclfFile).Error
}

// "The MBI has 11 characters, like the Health Insurance Claim Number (HICN), which can have up to 11."
// https://www.cms.gov/Medicare/New-Medicare-Card/Understanding-the-MBI-with-Format.pdf
type CCLFBeneficiary struct {
	gorm.Model
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

func (suppressionFile *SuppressionFile) Delete() error {
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	err := db.Unscoped().Where("file_id = ?", suppressionFile.ID).Delete(&Suppression{}).Error
	if err != nil {
		return err
	}
	return db.Unscoped().Delete(&suppressionFile).Error
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
