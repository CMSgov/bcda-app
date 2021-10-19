package service

import (
	"context"
	goerrors "errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
)

type RequestConditions struct {
	ReqType   RequestType
	Resources []string

	CMSID string
	ACOID uuid.UUID

	JobID           uint
	Since           time.Time
	TransactionTime time.Time
	CreationTime    time.Time

	// Fields set in the service
	fileType models.CCLFFileType

	timeConstraint
}

type timeConstraint struct {
	attributionDate time.Time
	optOutDate      time.Time
	claimsDate      time.Time
}

type RequestType uint8

const (
	DefaultRequest          RequestType = iota
	RetrieveNewBeneHistData             // Allows caller to retrieve all of the data for newly attributed beneficiaries
	Runout                              // Allows caller to retrieve claims data for beneficiaries no longer attributed to the ACO
)

// Ensure service satisfies the interface
var _ Service = &service{}

// Service contains all of the methods needed to interact with the data represented in the models package
type Service interface {
	GetQueJobs(ctx context.Context, conditions RequestConditions) (queJobs []*models.JobEnqueueArgs, err error)

	GetAlrJobs(ctx context.Context, alrMBI *models.AlrMBIs) []*models.JobAlrEnqueueArgs

	GetJobAndKeys(ctx context.Context, jobID uint) (*models.Job, []*models.JobKey, error)

	GetJobs(ctx context.Context, acoID uuid.UUID, statuses ...models.JobStatus) ([]*models.Job, error)

	CancelJob(ctx context.Context, jobID uint) (uint, error)

	GetJobPriority(acoID string, resourceType string, sinceParam bool) int16

	GetLatestCCLFFile(ctx context.Context, cmsID string, fileType models.CCLFFileType) (*models.CCLFFile, error)

	GetACOConfigForID(cmsID string) (*ACOConfig, bool)
}

const (
	cclf8FileNum = int(8)
)

func NewService(r models.Repository, cfg *Config, basePath string) Service {
	acoMap := make(map[*regexp.Regexp]*ACOConfig)
	for idx := range cfg.ACOConfigs {
		acoCfg := cfg.ACOConfigs[idx]
		acoMap[acoCfg.patternExp] = &acoCfg
	}

	return &service{
		repository:        r,
		logger:            log.API,
		stdCutoffDuration: cfg.cutoffDuration,
		sp: suppressionParameters{
			includeSuppressedBeneficiaries: false,
			lookbackDays:                   cfg.SuppressionLookbackDays,
		},
		rp: runoutParameters{
			// Runouts apply to claims data for the previous year.
			claimThruDate:  cfg.RunoutConfig.claimThru,
			cutoffDuration: cfg.RunoutConfig.cutoffDuration,
		},
		bbBasePath:    basePath,
		acoConfig:     acoMap,
		alrMBIsPerJob: cfg.AlrJobSize,
	}
}

type service struct {
	repository models.Repository

	logger logrus.FieldLogger

	stdCutoffDuration time.Duration
	sp                suppressionParameters
	rp                runoutParameters
	bbBasePath        string

	// Links pattern match to the associated ACO config
	acoConfig map[*regexp.Regexp]*ACOConfig

	alrMBIsPerJob uint
}

type suppressionParameters struct {
	includeSuppressedBeneficiaries bool
	lookbackDays                   int
}

type runoutParameters struct {
	// All claims data occur at or before this date
	claimThruDate time.Time
	// Amount of time the callers can retrieve runout data (relative to when runout data was ingested)
	cutoffDuration time.Duration
}

func (s *service) GetQueJobs(ctx context.Context, conditions RequestConditions) (queJobs []*models.JobEnqueueArgs, err error) {
	if conditions.timeConstraint, err = s.timeConstraints(ctx, conditions.CMSID); err != nil {
		return nil, fmt.Errorf("failed to set time constraints for caller: %w", err)
	}

	var (
		beneficiaries, newBeneficiaries []*models.CCLFBeneficiary
		jobs                            []*models.JobEnqueueArgs
	)

	if conditions.ReqType == Runout {
		conditions.fileType = models.FileTypeRunout
	} else {
		conditions.fileType = models.FileTypeDefault
	}

	hasAttributionDate := !conditions.attributionDate.IsZero()

	// for default requests, runouts, or any requests where the Since parameter is
	// after a terminated ACO's attribution date, we should only retrieve exisiting benes
	if conditions.ReqType == DefaultRequest ||
		conditions.ReqType == Runout ||
		hasAttributionDate && conditions.Since.After(conditions.attributionDate) {

		beneficiaries, err = s.getBeneficiaries(ctx, conditions)
		if err != nil {
			return nil, err
		}
	} else if conditions.ReqType == RetrieveNewBeneHistData {
		newBeneficiaries, beneficiaries, err = s.getNewAndExistingBeneficiaries(ctx, conditions)
		if err != nil {
			return nil, err
		}
		// add new beneficiaries to the job queue; use a default time value to ensure
		// that we retrieve the full history for these beneficiaries
		jobs, err = s.createQueueJobs(conditions, time.Time{}, newBeneficiaries)
		if err != nil {
			return nil, err
		}
		queJobs = append(queJobs, jobs...)
	} else {
		return nil, fmt.Errorf("Unsupported RequestType %d", conditions.ReqType)
	}

	// add existiing beneficiaries to the job queue
	jobs, err = s.createQueueJobs(conditions, conditions.Since, beneficiaries)
	if err != nil {
		return nil, err
	}

	queJobs = append(queJobs, jobs...)

	return queJobs, nil
}

func (s *service) GetJobAndKeys(ctx context.Context, jobID uint) (*models.Job, []*models.JobKey, error) {
	j, err := s.repository.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, nil, err
	}

	// No need to look up job keys if the job is complete
	if j.Status != models.JobStatusCompleted {
		return j, nil, nil
	}

	keys, err := s.repository.GetJobKeys(ctx, jobID)
	if err != nil {
		return nil, nil, err
	}

	nonEmptyKeys := make([]*models.JobKey, 0, len(keys))
	for i, key := range keys {
		if strings.TrimSpace(key.FileName) == models.BlankFileName {
			continue
		}
		nonEmptyKeys = append(nonEmptyKeys, keys[i])
	}

	return j, nonEmptyKeys, nil
}

func (s *service) GetJobs(ctx context.Context, acoID uuid.UUID, statuses ...models.JobStatus) ([]*models.Job, error) {
	jobs, err := s.repository.GetJobs(ctx, acoID, statuses...)
	if err != nil {
		return nil, err
	}

	if jobs == nil {
		return nil, JobsNotFoundError{acoID, statuses}
	}
	return jobs, nil
}

type JobsNotFoundError struct {
	ACOID       uuid.UUID
	StatusTypes []models.JobStatus
}

func (e JobsNotFoundError) Error() string {
	return fmt.Sprintf("no Jobs found for acoID %s with job statuses %s",
		e.ACOID, e.StatusTypes)
}

func (s *service) CancelJob(ctx context.Context, jobID uint) (uint, error) {
	// Assumes the job exists and retrieves the job by ID
	job, err := s.repository.GetJobByID(ctx, jobID)
	if err != nil {
		return 0, err
	}

	// Check if the job is pending or in progress.
	if job.Status == models.JobStatusPending || job.Status == models.JobStatusInProgress {
		job.Status = models.JobStatusCancelled
		err = s.repository.UpdateJob(ctx, *job)
		if err != nil {
			return 0, ErrJobNotCancelled
		}
		return jobID, nil
	}

	// Return 0, ErrJobNotCancellable when attempting to cancel a non-cancellable job.
	return 0, ErrJobNotCancellable
}

func (s *service) createQueueJobs(conditions RequestConditions, since time.Time, beneficiaries []*models.CCLFBeneficiary) (jobs []*models.JobEnqueueArgs, err error) {

	// persist in format ready for usage with _lastUpdated -- i.e., prepended with 'gt'
	var sinceArg string
	if !since.IsZero() {
		sinceArg = "gt" + since.Format(time.RFC3339Nano)
	}

	for _, rt := range conditions.Resources {
		maxBeneficiaries, err := getMaxBeneCount(rt)
		if err != nil {
			return nil, err
		}

		var rowCount = 0
		jobIDs := make([]string, 0, maxBeneficiaries)
		for _, b := range beneficiaries {
			rowCount++
			jobIDs = append(jobIDs, fmt.Sprint(b.ID))
			if len(jobIDs) >= maxBeneficiaries || rowCount >= len(beneficiaries) {
				if acoConfig, ok := s.GetACOConfigForID(conditions.CMSID); ok {
					// Create separate jobs for each data type if needed
					for _, dataType := range acoConfig.Data {
						// conditions.TransactionTime references the last time adjudicated data
						// was updated in the BB client. If we are queuing up a pre-adjudicated
						// data job, we need to assume that the adjudicated and pre-adjudicated
						// data ingestion timelines don't line up, therefore for all
						// pre-adjudicated jobs we will just use conditions.CreationTime as an
						// upper bound
						var transactionTime time.Time
						if dataType == constants.PreAdjudicated {
							transactionTime = conditions.CreationTime
						} else {
							transactionTime = conditions.TransactionTime
						}
						if resource, ok := GetDataType(rt); ok {
							if resource.SupportsDataType(dataType) {
								enqueueArgs := models.JobEnqueueArgs{
									ID:              int(conditions.JobID),
									ACOID:           conditions.ACOID.String(),
									BeneficiaryIDs:  jobIDs,
									ResourceType:    rt,
									Since:           sinceArg,
									TransactionTime: transactionTime,
									BBBasePath:      s.bbBasePath,
									DataType:        dataType,
								}

								s.setClaimsDate(&enqueueArgs, conditions)

								jobs = append(jobs, &enqueueArgs)
							}
						} else {
							// This should never be possible, would have returned earlier
							return nil, errors.New("Invalid resource type: " + rt)
						}
					}

					jobIDs = make([]string, 0, maxBeneficiaries)
				} else {
					// This should never be possible, would have returned earlier
					return nil, errors.New("Invalid ACO")
				}
			}
		}
	}

	return jobs, nil
}

func (s *service) getNewAndExistingBeneficiaries(ctx context.Context, conditions RequestConditions) (newBeneficiaries, beneficiaries []*models.CCLFBeneficiary, err error) {

	var (
		cutoffTime time.Time
	)

	// only set cutoffTime if there is no attributionDate set
	if s.stdCutoffDuration > 0 && conditions.attributionDate.IsZero() {
		cutoffTime = time.Now().Add(-1 * s.stdCutoffDuration)
	}

	// will get all benes between cutoff time and now, or all benes up until the attribution date
	cclfFileNew, err := s.repository.GetLatestCCLFFile(ctx, conditions.CMSID, cclf8FileNum, constants.ImportComplete,
		cutoffTime, conditions.attributionDate, conditions.fileType)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get new CCLF file for cmsID %s %s", conditions.CMSID, err.Error())
	}
	if cclfFileNew == nil {
		return nil, nil, CCLFNotFoundError{8, conditions.CMSID, conditions.fileType, cutoffTime}
	}

	cclfFileOld, err := s.repository.GetLatestCCLFFile(ctx, conditions.CMSID, cclf8FileNum, constants.ImportComplete,
		time.Time{}, conditions.Since, models.FileTypeDefault)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get old CCLF file for cmsID %s %s", conditions.CMSID, err.Error())
	}

	if cclfFileOld == nil {
		s.logger.Infof("Unable to find CCLF8 File for cmsID %s prior to date: %s; all beneficiaries will be considered NEW",
			conditions.CMSID, conditions.Since)
		newBeneficiaries, err = s.getBenesByFileID(ctx, cclfFileNew.ID, conditions)
		if err != nil {
			return nil, nil, err
		}
		if len(newBeneficiaries) == 0 {
			return nil, nil, fmt.Errorf("Found 0 new beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d",
				conditions.CMSID, cclfFileNew.ID)
		}
		return newBeneficiaries, nil, nil
	}

	oldMBIs, err := s.repository.GetCCLFBeneficiaryMBIs(ctx, cclfFileOld.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve MBIs for cmsID %s cclfFileID %d %s",
			conditions.CMSID, cclfFileOld.ID, err.Error())
	}

	// Retrieve all of the benes associated with this CCLF file.
	benes, err := s.getBenesByFileID(ctx, cclfFileNew.ID, conditions)
	if err != nil {
		return nil, nil, err
	}
	if len(benes) == 0 {
		return nil, nil, fmt.Errorf("Found 0 new or existing beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d",
			conditions.CMSID, cclfFileNew.ID)
	}

	// Split the results beteween new and old benes based on the existence of the bene in the old map
	oldMBIMap := make(map[string]struct{}, len(oldMBIs))
	for _, oldMBI := range oldMBIs {
		oldMBIMap[oldMBI] = struct{}{}
	}
	for _, bene := range benes {
		if _, ok := oldMBIMap[bene.MBI]; ok {
			beneficiaries = append(beneficiaries, bene)
		} else {
			newBeneficiaries = append(newBeneficiaries, bene)
		}
	}

	return newBeneficiaries, beneficiaries, nil
}

func (s *service) getBeneficiaries(ctx context.Context, conditions RequestConditions) ([]*models.CCLFBeneficiary, error) {
	var (
		cutoffTime time.Time
	)

	// only set a cutoffTime if there is no attributionDate set
	if conditions.attributionDate.IsZero() {
		if conditions.fileType == models.FileTypeDefault && s.stdCutoffDuration > 0 {
			cutoffTime = time.Now().Add(-1 * s.stdCutoffDuration)
		} else if conditions.fileType == models.FileTypeRunout && s.rp.cutoffDuration > 0 {
			cutoffTime = time.Now().Add(-1 * s.rp.cutoffDuration)
		}
	}
	cclfFile, err := s.repository.GetLatestCCLFFile(ctx, conditions.CMSID, cclf8FileNum,
		constants.ImportComplete, cutoffTime, conditions.attributionDate, conditions.fileType)
	if err != nil {
		return nil, fmt.Errorf("failed to get CCLF file for cmsID %s fileType %d %s",
			conditions.CMSID, conditions.fileType, err.Error())
	}
	if cclfFile == nil {
		return nil, CCLFNotFoundError{8, conditions.CMSID, conditions.fileType, cutoffTime}
	}

	benes, err := s.getBenesByFileID(ctx, cclfFile.ID, conditions)
	if err != nil {
		return nil, err
	}
	if len(benes) == 0 {
		return nil, fmt.Errorf("Found 0 beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d",
			conditions.CMSID, cclfFile.ID)
	}

	return benes, nil
}

func (s *service) getBenesByFileID(ctx context.Context, cclfFileID uint, conditions RequestConditions) ([]*models.CCLFBeneficiary, error) {
	var (
		ignoredMBIs []string
		err         error
	)
	if !s.sp.includeSuppressedBeneficiaries {
		upperBound := conditions.optOutDate
		if conditions.optOutDate.IsZero() {
			upperBound = time.Now()
		}

		ignoredMBIs, err = s.repository.GetSuppressedMBIs(ctx, s.sp.lookbackDays, upperBound)
		if err != nil {
			return nil, fmt.Errorf("failed to retreive suppressedMBIs %s", err.Error())
		}
	}

	benes, err := s.repository.GetCCLFBeneficiaries(ctx, cclfFileID, ignoredMBIs)
	if err != nil {
		return nil, fmt.Errorf("failed to get beneficiaries %s", err.Error())
	}

	return benes, nil
}

// timeConstraints searches for any time bounds that we should apply on the associated ACO
func (s *service) timeConstraints(ctx context.Context, cmsID string) (timeConstraint, error) {
	var constraint timeConstraint
	aco, err := s.repository.GetACOByCMSID(ctx, cmsID)
	if err != nil {
		return constraint, fmt.Errorf("failed to retrieve aco: %w", err)
	}

	// If aco is not terminated, then we should not apply any time constraints
	if aco.TerminationDetails == nil {
		return constraint, nil
	}

	constraint.attributionDate = aco.TerminationDetails.AttributionDate()
	constraint.claimsDate = aco.TerminationDetails.ClaimsDate()
	constraint.optOutDate = aco.TerminationDetails.OptOutDate()
	return constraint, nil
}

// setClaimsDate computes the claims window to apply on the args
func (s *service) setClaimsDate(args *models.JobEnqueueArgs, conditions RequestConditions) {

	// If the caller made a request for runout data
	// it takes precedence over any other claims date
	// that may be applied
	if conditions.ReqType == Runout {
		args.ClaimsWindow.UpperBound = s.rp.claimThruDate
	} else if !conditions.claimsDate.IsZero() {
		args.ClaimsWindow.UpperBound = conditions.claimsDate
	}

	for pattern, cfg := range s.acoConfig {
		if pattern.MatchString(conditions.CMSID) {
			args.ClaimsWindow.LowerBound = cfg.LookbackTime()
			break
		}
	}
}

// Gets the priority for the job where the lower the number the higher the priority in the queue.
// Priority is based on the request parameters that the job is executing on.
func (s *service) GetJobPriority(acoID string, resourceType string, sinceParam bool) int16 {
	var priority int16
	if isPriorityACO(acoID) {
		priority = int16(10) // priority level for jobs for synthetic ACOs that are used for smoke testing
	} else if resourceType == "Patient" || resourceType == "Coverage" {
		priority = int16(20) // priority level for jobs that only request smaller resources
	} else if sinceParam {
		priority = int16(30) // priority level for jobs that only request data for a limited timeframe
	} else {
		priority = int16(100) // default priority level for jobs
	}
	return priority
}

// GetACOConfigForID gets any currently loaded ACOConfig for the matching cmsID
func (s *service) GetACOConfigForID(cmsID string) (*ACOConfig, bool) {
	for pattern, cfg := range s.acoConfig {
		if pattern.MatchString(cmsID) {
			return cfg, true
		}
	}

	return nil, false
}

// Checks to see if an ACO is priority ACO based on a regex pattern provided by an
// environment variable.
func isPriorityACO(acoID string) bool {
	if priorityACOPattern := conf.GetEnv("PRIORITY_ACO_REG_EX"); priorityACOPattern != "" {
		var priorityACORegex = regexp.MustCompile(priorityACOPattern)
		if priorityACORegex.MatchString(acoID) {
			return true
		}
	}
	return false

}

func getMaxBeneCount(requestType string) (int, error) {
	const (
		BCDA_FHIR_MAX_RECORDS_EOB_DEFAULT           = 200
		BCDA_FHIR_MAX_RECORDS_PATIENT_DEFAULT       = 5000
		BCDA_FHIR_MAX_RECORDS_COVERAGE_DEFAULT      = 4000
		BCDA_FHIR_MAX_RECORDS_CLAIM_DEFAULT         = 4000
		BCDA_FHIR_MAX_RECORDS_CLAIMRESPONSE_DEFAULT = 4000
	)
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
	case "Claim":
		envVar = "BCDA_FHIR_MAX_RECORDS_CLAIM"
		defaultVal = BCDA_FHIR_MAX_RECORDS_CLAIM_DEFAULT
	case "ClaimResponse":
		envVar = "BCDA_FHIR_MAX_RECORDS_CLAIM_RESPONSE"
		defaultVal = BCDA_FHIR_MAX_RECORDS_CLAIMRESPONSE_DEFAULT
	default:
		err := errors.New("invalid request type")
		return -1, err
	}
	maxBeneficiaries := utils.GetEnvInt(envVar, defaultVal)

	return maxBeneficiaries, nil
}

// IsSupportedACO determines if the particular ACO is supported by checking
// its CMS_ID against the supported formats.
func IsSupportedACO(cmsID string) bool {
	const (
		ssp     = `^A\d{4}$`
		ngaco   = `^V\d{3}$`
		cec     = `^E\d{4}$`
		ckcc    = `^C\d{4}$`
		kcf     = `^K\d{4}$`
		dc      = `^D\d{4}$`
		test    = `^TEST\d{3}$`
		pattern = `(` + ssp + `)|(` + ngaco + `)|(` + cec + `)|(` + ckcc + `)|(` + kcf + `)|(` + dc + `)|(` + test + `)`
	)

	return regexp.MustCompile(pattern).MatchString(cmsID)
}

func (s *service) GetLatestCCLFFile(ctx context.Context, cmsID string, fileType models.CCLFFileType) (*models.CCLFFile, error) {
	cclfFile, err := s.repository.GetLatestCCLFFile(ctx, cmsID, cclf8FileNum, constants.ImportComplete, time.Time{}, time.Time{}, fileType)
	if err != nil {
		return nil, err
	}

	if cclfFile == nil {
		return nil, CCLFNotFoundError{8, cmsID, fileType, time.Time{}}
	}

	return cclfFile, nil
}

type CCLFNotFoundError struct {
	FileNumber int
	CMSID      string
	FileType   models.CCLFFileType
	CutoffTime time.Time
}

func (e CCLFNotFoundError) Error() string {
	return fmt.Sprintf("no CCLF%d file found for cmsID %s fileType %d cutoffTime %s",
		e.FileNumber, e.CMSID, e.FileType, e.CutoffTime.String())
}

var (
	ErrJobNotCancelled   = goerrors.New("Job was not cancelled due to internal server error.")
	ErrJobNotCancellable = goerrors.New("Job was not cancelled because it is not Pending or In Progress")
)
