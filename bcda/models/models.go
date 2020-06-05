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

	"github.com/CMSgov/bcda-app/bcda/constants"

	authclient "github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/auth/rsautils"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/bgentry/que-go"
	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const BCDA_FHIR_MAX_RECORDS_EOB_DEFAULT = 200
const BCDA_FHIR_MAX_RECORDS_PATIENT_DEFAULT = 5000
const BCDA_FHIR_MAX_RECORDS_COVERAGE_DEFAULT = 4000

func InitializeGormModels() *gorm.DB {
	log.Println("Initialize bcda models")
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	// Migrate the schema
	// Add your new models here
	// This should probably not be called in production
	// What happens when you need to make a database change, there's already data you need to preserve, and
	// you need to run a script to migrate existing data to its new home or shape?
	db.AutoMigrate(
		&ACO{},
		&Job{},
		&JobKey{},
		&CCLFBeneficiaryXref{},
		&CCLFFile{},
		&CCLFBeneficiary{},
		&Suppression{},
		&SuppressionFile{},
	)

	db.Model(&CCLFBeneficiary{}).AddForeignKey("file_id", "cclf_files(id)", "RESTRICT", "RESTRICT")

	return db
}

type Job struct {
	gorm.Model
	ACO               ACO       `gorm:"foreignkey:ACOID;association_foreignkey:UUID"` // aco
	ACOID             uuid.UUID `gorm:"type:char(36)" json:"aco_id"`
	RequestURL        string    `json:"request_url"` // request_url
	Status            string    `json:"status"`      // status
	TransactionTime   time.Time // most recent data load transaction time from BFD
	JobCount          int
	CompletedJobCount int
	JobKeys           []JobKey
}

func (job *Job) CheckCompletedAndCleanup(db *gorm.DB) (bool, error) {

	// Trivial case, no need to keep going
	if job.Status == "Completed" {
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
		return true, db.Model(&job).Update("status", "Completed").Error
	}

	return false, nil
}

func (job *Job) GetEnqueJobs(resourceTypes []string, since string, newBeneficiariesOnly bool) (enqueJobs []*que.Job, err error) {
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	var aco ACO
	var beneficiaries []CCLFBeneficiary
	err = db.Find(&aco, "uuid = ?", job.ACOID).Error
	if err != nil {
		return nil, err
	}

	// only retrieve new beneficiaries (comparison of current CCLF vs previous CCLF)
	if newBeneficiariesOnly {
		// includeSuppressed = false to exclude beneficiaries who have opted out of data sharing
		beneficiaries, err = aco.GetNewBeneficiaries(false)
		if err != nil {
			return nil, err
		}
	} else {
		// includeSuppressed = false to exclude beneficiaries who have opted out of data sharing
		beneficiaries, err = aco.GetBeneficiaries(false)
		if err != nil {
			return nil, err
		}
	}

	for _, rt := range resourceTypes {
		var rowCount = 0
		var jobIDs []string
		maxBeneficiaries, err := GetMaxBeneCount(rt)
		if err != nil {
			return nil, err
		}
		for _, b := range beneficiaries {
			rowCount++
			jobIDs = append(jobIDs, fmt.Sprint(b.ID))
			if len(jobIDs) >= maxBeneficiaries || rowCount >= len(beneficiaries) {

				args, err := json.Marshal(jobEnqueueArgs{
					ID:              int(job.ID),
					ACOID:           job.ACOID.String(),
					BeneficiaryIDs:  jobIDs,
					ResourceType:    rt,
					Since:           since,
					TransactionTime: job.TransactionTime,
					Priority:        setJobPriority(*aco.CMSID, rt, len(since) != 0),
				})
				if err != nil {
					return nil, err
				}

				j := &que.Job{
					Type: "ProcessJob",
					Args: args,
				}

				enqueJobs = append(enqueJobs, j)

				jobIDs = []string{}
			}
		}
	}
	return enqueJobs, nil
}

// Sets the priority for the job where the lower the number the higher the priority in the queue.
// Prioirity is based on the request parameters that the job is executing on.
func setJobPriority(acoID string, resourceType string, sinceParam bool) int {
	var priority int
	if isPriorityACO(acoID) {
		priority = 10 // priority level for jobs for sythetic ACOs that are used for smoke testing
	} else if resourceType == "Patient" || resourceType == "Coverage" {
		priority = 20 // priority level for jobs that only request smaller resources
	} else if sinceParam {
		priority = 30 // priority level for jobs that only request data for a limited timeframe
	} else {
		priority = 100 // default priority level for jobs
	}
	return priority
}

// Checks to see if an ACO is priority ACO based on a list provided by an
// environment variable.
func isPriorityACO(acoID string) bool {
	if priorityACOList := os.Getenv("PRIORITY_ACO_IDS"); priorityACOList != "" {
		priorityACOs := strings.Split(priorityACOList, ",")
		for _, priorityACO := range priorityACOs {
			if priorityACO == acoID {
				return true
			}
		}
	}
	return false
}

func (j *Job) StatusMessage() string {
	if j.Status == "In Progress" && j.JobCount > 0 {
		pct := float64(j.CompletedJobCount) / float64(j.JobCount) * 100
		return fmt.Sprintf("%s (%d%%)", j.Status, int(pct))
	}

	return j.Status
}

func GetMaxBeneCount(requestType string) (int, error) {
	var envVar string
	var defaultVal int

	switch requestType {
	case "ExplanationOfBenefit":
		envVar = "BCDA_FHIR_MAX_RECORDS_EOB"
		defaultVal = BCDA_FHIR_MAX_RECORDS_EOB_DEFAULT
	case "Patient":
		envVar = "BCDA_FHIR_MAX_RECORDS_PATIENT"
		defaultVal = BCDA_FHIR_MAX_RECORDS_PATIENT_DEFAULT
	case "Coverage":
		envVar = "BCDA_FHIR_MAX_RECORDS_COVERAGE"
		defaultVal = BCDA_FHIR_MAX_RECORDS_COVERAGE_DEFAULT
	default:
		err := errors.New("invalid request type")
		return -1, err
	}
	maxBeneficiaries := utils.GetEnvInt(envVar, defaultVal)

	return maxBeneficiaries, nil
}

type JobKey struct {
	gorm.Model
	Job          Job    `gorm:"foreignkey:jobID"`
	JobID        uint   `gorm:"primary_key" json:"job_id"`
	FileName     string `gorm:"type:char(127)"`
	ResourceType string
}

// ACO represents an Accountable Care Organization.
type ACO struct {
	gorm.Model
	UUID        uuid.UUID `gorm:"primary_key;type:char(36)" json:"uuid"`
	CMSID       *string   `gorm:"type:char(5);unique" json:"cms_id"`
	Name        string    `json:"name"`
	ClientID    string    `json:"client_id"`
	GroupID     string    `json:"group_id"`
	SystemID    string    `json:"system_id"`
	AlphaSecret string    `json:"alpha_secret"`
	PublicKey   string    `json:"public_key"`
}

// GetNewBeneficiaries retrieves new beneficiaries associated with the ACO
// "new" is defined as beneficiaries existing in current CCLF that did not exist in previous CCLF
func (aco *ACO) GetNewBeneficiaries(includeSuppressed bool) ([]CCLFBeneficiary, error) {
	var cclfBeneficiaries []CCLFBeneficiary
	var cclfBeneficiariesOld []string

	if aco.CMSID == nil {
		log.Errorf("No CMSID set for ACO: %s", aco.UUID)
		return cclfBeneficiaries, fmt.Errorf("no CMS ID set for this ACO")
	}
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	var cclfFileNew CCLFFile
	var cclfFileOld CCLFFile
	// todo add a filter here to make sure the file is up to date.
	if db.Where("aco_cms_id = ? and cclf_num = 8 and import_status= ?", aco.CMSID, constants.ImportComplete).Order("timestamp desc").First(&cclfFileNew).RecordNotFound() {
		log.Errorf("Unable to find new CCLF8 File for ACO: %v", *aco.CMSID)
		return cclfBeneficiaries, fmt.Errorf("unable to find new cclfFile")
	}
	// Offset(1) is used to get skip the most recent file, and get the one before it
	// TODO: later, once Group/new is out of test/research phase, we can make this smarter based on feedback
	if db.Where("aco_cms_id = ? and cclf_num = 8 and import_status= ?", aco.CMSID, constants.ImportComplete).Order("timestamp desc").Offset(1).First(&cclfFileOld).RecordNotFound() {
		log.Errorf("Unable to find last month's CCLF8 File for ACO: %v", *aco.CMSID)
		return cclfBeneficiaries, fmt.Errorf("unable to find last month's cclfFile")
	}

	var suppressedBBIDs []string

	if !includeSuppressed {
		suppressedBBIDs = GetSuppressedBlueButtonIDs(db)
	}

	// Get all the beneficiaries from the previous CCLF for this ACO, to use as a basis for comparison
	var err error
	err = db.Table("cclf_beneficiaries").Where("file_id = ?", cclfFileOld.ID).Pluck("mbi", &cclfBeneficiariesOld).Error
	if err != nil {
		log.Errorf("Error retrieving beneficiaries from previous CCLF8 for ACO ID %s: %s", aco.UUID.String(), err.Error())
		return nil, err
	}

	if suppressedBBIDs != nil {
		err = db.Not("blue_button_id", suppressedBBIDs).Not("mbi", cclfBeneficiariesOld).Find(&cclfBeneficiaries, "file_id = ?", cclfFileNew.ID).Error
	} else {
		err = db.Not("mbi", cclfBeneficiariesOld).Find(&cclfBeneficiaries, "file_id = ?", cclfFileNew.ID).Error
	}

	if err != nil {
		log.Errorf("Error retrieving new beneficiaries from CCLF8 for ACO ID %s: %s", aco.UUID.String(), err.Error())
		return nil, err
	} else if len(cclfBeneficiaries) == 0 {
		log.Errorf("Found 0 new beneficiaries from CCLF8 file for ACO ID %s", aco.UUID.String())
		return nil, fmt.Errorf("found 0 new  beneficiaries from CCLF8 for ACO ID %s", aco.UUID.String())
	}

	return cclfBeneficiaries, nil
}

// GetBeneficiaries retrieves all beneficiaries associated with the ACO.
func (aco *ACO) GetBeneficiaries(includeSuppressed bool) ([]CCLFBeneficiary, error) {
	var cclfBeneficiaries []CCLFBeneficiary

	if aco.CMSID == nil {
		log.Errorf("No CMSID set for ACO: %s", aco.UUID)
		return cclfBeneficiaries, fmt.Errorf("no CMS ID set for this ACO")
	}
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	var cclfFile CCLFFile
	// todo add a filter here to make sure the file is up to date.
	if db.Where("aco_cms_id = ? and cclf_num = 8 and import_status= ?", aco.CMSID, constants.ImportComplete).Order("timestamp desc").First(&cclfFile).RecordNotFound() {
		log.Errorf("Unable to find CCLF8 File for ACO: %v", *aco.CMSID)
		return cclfBeneficiaries, fmt.Errorf("unable to find cclfFile")
	}

	var suppressedBBIDs []string

	if !includeSuppressed {
		suppressedBBIDs = GetSuppressedBlueButtonIDs(db)
	}

	var err error
	if suppressedBBIDs != nil {
		err = db.Not("blue_button_id", suppressedBBIDs).Find(&cclfBeneficiaries, "file_id = ?", cclfFile.ID).Error
	} else {
		err = db.Find(&cclfBeneficiaries, "file_id = ?", cclfFile.ID).Error
	}

	if err != nil {
		log.Errorf("Error retrieving beneficiaries from latest CCLF8 file for ACO ID %s: %s", aco.UUID.String(), err.Error())
		return nil, err
	} else if len(cclfBeneficiaries) == 0 {
		log.Errorf("Found 0 beneficiaries from latest CCLF8 file for ACO ID %s", aco.UUID.String())
		return nil, fmt.Errorf("found 0 beneficiaries from latest CCLF8 file for ACO ID %s", aco.UUID.String())
	}

	return cclfBeneficiaries, nil
}

func GetSuppressedBlueButtonIDs(db *gorm.DB) []string {

	var suppressedBBIDs []string

	// Set the window that the suppression query should examine. Will default to 60 if no value is provided
	suppressionLookback := strconv.Itoa(utils.GetEnvInt("BCDA_SUPPRESSION_LOOKBACK_DAYS", 60))

	// #nosec G202
	suppressionQuery := `SELECT DISTINCT s.blue_button_id
	FROM (
		SELECT blue_button_id, MAX(effective_date) max_date
		FROM suppressions
		WHERE (NOW() - interval '` + suppressionLookback + ` days') < effective_date AND effective_date <= NOW()
					AND preference_indicator != '' AND blue_button_id != '' AND blue_button_id IS NOT NULL
		GROUP BY blue_button_id
	) h
	JOIN suppressions s ON s.blue_button_id = h.blue_button_id and s.effective_date = h.max_date
	WHERE preference_indicator = 'N'`
	db.Raw(suppressionQuery).Pluck("blue_button_id", &suppressedBBIDs)

	return suppressedBBIDs
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

type CCLFFile struct {
	gorm.Model
	CCLFNum         int       `gorm:"not null"`
	Name            string    `gorm:"not null;unique"`
	ACOCMSID        string    `gorm:"column:aco_cms_id"`
	Timestamp       time.Time `gorm:"not null"`
	PerformanceYear int       `gorm:"not null"`
	ImportStatus    string    `gorm:"column:import_status"`
}

func (cclfFile *CCLFFile) Delete() error {
	db := database.GetGORMDbConnection()
	defer db.Close()
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
	CCLFFile     CCLFFile
	FileID       uint   `gorm:"not null;index:idx_cclf_beneficiaries_file_id"`
	HICN         string `gorm:"type:varchar(11);not null;index:idx_cclf_beneficiaries_hicn"`
	MBI          string `gorm:"type:char(11);not null;index:idx_cclf_beneficiaries_mbi"`
	BlueButtonID string `gorm:"type: text;index:idx_cclf_beneficiaries_bb_id"`
}

type SuppressionFile struct {
	gorm.Model
	Name         string    `gorm:"not null;unique"`
	Timestamp    time.Time `gorm:"not null"`
	ImportStatus string    `gorm:"column:import_status"`
}

func (suppressionFile *SuppressionFile) Delete() error {
	db := database.GetGORMDbConnection()
	defer db.Close()
	err := db.Unscoped().Where("file_id = ?", suppressionFile.ID).Delete(&Suppression{}).Error
	if err != nil {
		return err
	}
	return db.Unscoped().Delete(&suppressionFile).Error
}

type Suppression struct {
	gorm.Model
	SuppressionFile     SuppressionFile
	FileID              uint      `gorm:"not null"`
	BlueButtonID        string    `gorm:"type: text;index:idx_suppression_bb_id"`
	MBI                 string    `gorm:"type:varchar(11)"`
	HICN                string    `gorm:"type:varchar(11)"`
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

	modelIdentifier := cclfBeneficiary.HICN
	patientIdMode := utils.FromEnv("PATIENT_IDENTIFIER_MODE", "HICN_MODE")
	if patientIdMode == "MBI_MODE" {
		modelIdentifier = cclfBeneficiary.MBI
	}

	blueButtonID, err = GetBlueButtonID(bb, modelIdentifier, patientIdMode, "beneficiary", cclfBeneficiary.ID)
	if err != nil {
		return "", err
	}
	return blueButtonID, nil
}

// This method will ensure that a valid BlueButton ID is returned.
// If you use suppressionBeneficiary.BlueButtonID you will not be guaranteed a valid value
func (suppressionBeneficiary *Suppression) GetBlueButtonID(bb client.APIClient) (blueButtonID string, err error) {

	// support legacy data (before March 2020) when NGD only supported HICN
	modelIdentifier := suppressionBeneficiary.HICN
	patientIdMode := "HICN_MODE"

	// NGD added support for MBI in March 2020
	if modelIdentifier == "" {
		modelIdentifier = suppressionBeneficiary.MBI
		patientIdMode = "MBI_MODE"
	}

	blueButtonID, err = GetBlueButtonID(bb, modelIdentifier, patientIdMode, "suppression", suppressionBeneficiary.ID)
	if err != nil {
		return "", err
	}
	return blueButtonID, nil
}

func GetBlueButtonID(bb client.APIClient, modelIdentifier, patientIdMode, reqType string, modelID uint) (blueButtonID string, err error) {
	hashedIdentifier := client.HashIdentifier(modelIdentifier)

	// until NGD supports MBI, pass in the patientIdMode
	jsonData, err := bb.GetPatientByIdentifierHash(hashedIdentifier, patientIdMode)
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
			if patientIdMode == "MBI_MODE" {
				if identifier.Value == modelIdentifier {
					foundIdentifier = true
				}
			} else if patientIdMode == "HICN_MODE" {
				// no value to check against, should be removed when NGD supports MBI
				foundIdentifier = true
			}
		} else if strings.Contains(identifier.System, "bene_id") {
			if identifier.Value == blueButtonID {
				foundBlueButtonID = true
			}
		}
	}
	if !foundIdentifier {
		patientIdMode = strings.Split(patientIdMode, "_")[0]
		err = fmt.Errorf("%v not found in the identifiers", patientIdMode)
		log.Error(err)
		return "", err
	}
	if !foundBlueButtonID {
		err = fmt.Errorf("blue Button identifier not found in the identifiers")
		log.Error(err)
		return "", err
	}

	return blueButtonID, nil
}

// StoreSuppressionBBID stores the suppression beneficiary's Blue Button ID
// the ID value is retrieved from BB and saved.
func StoreSuppressionBBID() (success, failure int, err error) {
	db := database.GetGORMDbConnection()
	defer func() {
		err := db.Close()
		if err != nil {
			log.Error(err)
			return
		}
	}()

	bb, err := client.NewBlueButtonClient()
	if err != nil {
		err = errors.Wrap(err, "could not create Blue Button client")
		log.Error(err)
		return 0, 0, err
	}

	var suppressList []Suppression
	db.Find(&suppressList)
	for _, bene := range suppressList {
		suppressBene := bene
		bbID, err := suppressBene.GetBlueButtonID(bb)
		if err != nil {
			failure++
			continue
		}
		suppressBene.BlueButtonID = bbID
		db.Save(&suppressBene)
		success++
	}
	return success, failure, nil
}

// This is not a persistent model so it is not necessary to include in GORM auto migrate.
// swagger:ignore
type jobEnqueueArgs struct {
	ID              int
	ACOID           string
	BeneficiaryIDs  []string
	ResourceType    string
	Since           string
	TransactionTime time.Time
	Priority        int
}
