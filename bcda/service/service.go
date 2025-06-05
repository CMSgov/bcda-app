package service

import (
	"context"
	goerrors "errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ccoveille/go-safecast"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/CMSgov/bcda-app/middleware"
)

// Ensure service satisfies the interface
var _ Service = &service{}

// Service contains all of the methods needed to interact with the data represented in the models package
type Service interface {
	GetCutoffTime(ctx context.Context, reqType constants.DataRequestType, since time.Time, timeConstraints TimeConstraints, fileType models.CCLFFileType) (time.Time, string)
	FindOldCCLFFile(ctx context.Context, cmsID string, since time.Time, cclfTimestamp time.Time) (uint, error)
	GetQueJobs(ctx context.Context, args worker_types.PrepareJobArgs) (queJobs []*worker_types.JobEnqueueArgs, err error)
	GetAlrJobs(ctx context.Context, alrMBI *models.AlrMBIs) []*models.JobAlrEnqueueArgs
	GetJobAndKeys(ctx context.Context, jobID uint) (*models.Job, []*models.JobKey, error)
	GetJobKey(ctx context.Context, jobID uint, filename string) (*models.JobKey, error)
	GetJobs(ctx context.Context, acoID uuid.UUID, statuses ...models.JobStatus) ([]*models.Job, error)
	CancelJob(ctx context.Context, jobID uint) (uint, error)
	GetJobPriority(acoID string, resourceType string, sinceParam bool) int16
	GetLatestCCLFFile(ctx context.Context, cmsID string, lowerBound time.Time, upperBound time.Time, fileType models.CCLFFileType) (*models.CCLFFile, error)
	GetACOConfigForID(cmsID string) (*ACOConfig, bool)
	GetTimeConstraints(ctx context.Context, cmsID string) (TimeConstraints, error)
}

type service struct {
	repository models.Repository

	logger logrus.FieldLogger

	stdCutoffDuration time.Duration
	sp                suppressionParameters
	rp                runoutParameters
	bbBasePath        string

	// These are always searched in order and first matching config is used for any given ACO.
	acoConfigs []ACOConfig

	alrMBIsPerJob uint
}

func NewService(r models.Repository, cfg *Config, basePath string) Service {
	return &service{
		repository:        r,
		logger:            log.API,
		stdCutoffDuration: cfg.CutoffDuration,
		sp: suppressionParameters{
			includeSuppressedBeneficiaries: false,
			lookbackDays:                   cfg.SuppressionLookbackDays,
		},
		rp: runoutParameters{
			// Runouts apply to claims data for the previous year.
			claimThruDate:  cfg.RunoutConfig.claimThru,
			CutoffDuration: cfg.RunoutConfig.CutoffDuration,
		},
		bbBasePath:    basePath,
		acoConfigs:    cfg.ACOConfigs,
		alrMBIsPerJob: cfg.AlrJobSize,
	}
}

type TimeConstraints struct {
	AttributionDate time.Time
	OptOutDate      time.Time
	ClaimsDate      time.Time
}

type suppressionParameters struct {
	includeSuppressedBeneficiaries bool
	lookbackDays                   int
}

type runoutParameters struct {
	// All claims data occur at or before this date
	claimThruDate time.Time
	// Amount of time the callers can retrieve runout data (relative to when runout data was ingested)
	CutoffDuration time.Duration
}

func (s *service) GetCutoffTime(ctx context.Context, reqType constants.DataRequestType, since time.Time, timeConstraints TimeConstraints, fileType models.CCLFFileType) (cutoffTime time.Time, complexDataRequestType string) {
	hasAttributionDate := !timeConstraints.AttributionDate.IsZero()
	// for default requests, runouts, or any requests where the Since parameter is
	// after a terminated ACO's attribution date, we should only retrieve exisiting benes
	if reqType == constants.DefaultRequest ||
		reqType == constants.Runout ||
		(hasAttributionDate && since.After(timeConstraints.AttributionDate)) {
		complexDataRequestType = constants.GetExistingBenes
		// only set a cutoffTime if there is no attributionDate set
		if timeConstraints.AttributionDate.IsZero() {
			if fileType == models.FileTypeDefault && s.stdCutoffDuration > 0 {
				cutoffTime = time.Now().Add(-1 * s.stdCutoffDuration)
			} else if fileType == models.FileTypeRunout && s.rp.CutoffDuration > 0 {
				cutoffTime = time.Now().Add(-1 * s.rp.CutoffDuration)
			}
		}
	} else if reqType == constants.RetrieveNewBeneHistData {
		complexDataRequestType = constants.GetNewAndExistingBenes
		// only set cutoffTime if there is no attributionDate set
		if s.stdCutoffDuration > 0 && timeConstraints.AttributionDate.IsZero() {
			cutoffTime = time.Now().Add(-1 * s.stdCutoffDuration)
		}
	}

	return cutoffTime, complexDataRequestType
}

// FindOldCCLFFile finds an older CCLF file depending on passed in timestamps
func (s *service) FindOldCCLFFile(ctx context.Context, cmsID string, since time.Time, cclfTimestamp time.Time) (cclfFileOldID uint, err error) {
	// Retrieve an older CCLF file for beneficiary comparison.
	// This should be older than cclfFileNew AND prior to the "since" parameter, if provided.
	//
	// e.g.
	// - If it’s October 2023
	// - and they request all beneficiary data “since January 1st, 2023"
	// - any beneficiaries added in 2023 are considered "new."
	//
	oldFileUpperBound := since

	// If the _since parameter is more recent than the latest CCLF file timestamp, set the upper bound
	// for the older file to be prior to the newest file's timestamp.
	if !since.IsZero() && cclfTimestamp.Sub(since) < 0 {
		oldFileUpperBound = cclfTimestamp.Add(-1 * time.Second)
	}

	cclfFileOld, err := s.GetLatestCCLFFile(
		ctx,
		cmsID,
		time.Time{},
		oldFileUpperBound,
		models.FileTypeDefault,
	)
	if err != nil {
		return 0, err
	}

	if cclfFileOld == nil {
		s.logger.Infof("Unable to find CCLF8 File for cmsID %s prior to date: %s; all beneficiaries will be considered NEW", cmsID, since)
		return 0, nil
	} else {
		return cclfFileOld.ID, nil
	}
}

func (s *service) GetQueJobs(ctx context.Context, args worker_types.PrepareJobArgs) (queJobs []*worker_types.JobEnqueueArgs, err error) {
	var (
		beneficiaries, newBeneficiaries []*models.CCLFBeneficiary
		jobs                            []*worker_types.JobEnqueueArgs
	)

	// for default requests, runouts, or any requests where the Since parameter is
	// after a terminated ACO's attribution date, we should only retrieve exisiting benes
	switch args.ComplexDataRequestType {
	case constants.GetExistingBenes:
		beneficiaries, err = s.getBeneficiaries(ctx, args)
		if err != nil {
			return nil, err
		}
	case constants.GetNewAndExistingBenes:
		newBeneficiaries, beneficiaries, err = s.getNewAndExistingBeneficiaries(ctx, args)
		if err != nil {
			return nil, err
		}
		// add new beneficiaries to the job queue; use a default time value to ensure
		// that we retrieve the full history for these beneficiaries
		jobs, err = s.createQueueJobs(ctx, args, time.Time{}, newBeneficiaries)
		if err != nil {
			return nil, err
		}
		queJobs = append(queJobs, jobs...)
	default:
		return nil, fmt.Errorf("unsupported RequestType %d", args.RequestType)
	}

	// add existiing beneficiaries to the job queue
	jobs, err = s.createQueueJobs(ctx, args, args.Since, beneficiaries)
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

func (s *service) GetJobKey(ctx context.Context, jobID uint, filename string) (*models.JobKey, error) {
	return s.repository.GetJobKey(ctx, jobID, filename)
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

func (s *service) createQueueJobs(ctx context.Context, args worker_types.PrepareJobArgs, since time.Time, beneficiaries []*models.CCLFBeneficiary) (jobs []*worker_types.JobEnqueueArgs, err error) {
	// persist in format ready for usage with _lastUpdated -- i.e., prepended with 'gt'
	var sinceArg string
	if !since.IsZero() {
		sinceArg = "gt" + since.Format(time.RFC3339Nano)
	}

	acoCfg, ok := s.GetACOConfigForID(args.CMSID)
	if !ok {
		return nil, fmt.Errorf("failed to load or match ACO config (or potentially no ACO Configs set), CMS ID: %+v", args.CMSID)
	}

	for _, rt := range args.ResourceTypes {
		maxBeneficiaries, err := getMaxBeneCount(rt)
		if err != nil {
			return nil, err
		}

		rowCount := 0
		jobIDs := make([]string, 0, maxBeneficiaries)
		for _, b := range beneficiaries {
			rowCount++
			jobIDs = append(jobIDs, fmt.Sprint(b.ID))
			if len(jobIDs) >= maxBeneficiaries || rowCount >= len(beneficiaries) {
				// Create separate jobs for each data type if needed
				for _, dataType := range acoCfg.Data {
					// conditions.TransactionTime references the last time adjudicated data
					// was updated in the BB client. If we are queuing up a partially-adjudicated
					// data job, we need to assume that the adjudicated and partially-adjudicated
					// data ingestion timelines don't line up, therefore for all
					// partially-adjudicated jobs we will just use conditions.CreationTime as an
					// upper bound
					var transactionTime time.Time
					if dataType == constants.PartiallyAdjudicated {
						transactionTime = args.CreationTime
					} else {
						transactionTime = args.Job.TransactionTime
					}
					if resource, ok := GetDataType(rt); ok {
						if resource.SupportsDataType(dataType) {
							jobId, err := safecast.ToInt(args.Job.ID)
							if err != nil {
								log.API.Errorln(err)
							}
							enqueueArgs := worker_types.JobEnqueueArgs{
								ID:              jobId,
								ACOID:           args.ACOID.String(),
								CMSID:           args.CMSID,
								BeneficiaryIDs:  jobIDs,
								ResourceType:    rt,
								Since:           sinceArg,
								TransactionID:   ctx.Value(middleware.CtxTransactionKey).(string),
								TransactionTime: transactionTime,
								BBBasePath:      args.BFDPath,
								DataType:        dataType,
							}

							ok := s.setClaimsDate(&enqueueArgs, args)
							if !ok {
								return nil, fmt.Errorf("failed to load or match ACO config (or potentially no ACO Configs set), CMS ID: %+v", args.CMSID)
							}

							jobs = append(jobs, &enqueueArgs)
						}
					} else {
						// This should never be possible, would have returned earlier
						return nil, errors.New("Invalid resource type: " + rt)
					}
				}

				jobIDs = make([]string, 0, maxBeneficiaries)
			}
		}
	}

	return jobs, nil
}

// Returns the beneficiaries associated with the latest CCLF file for the given request conditions,
// split between existing beneficiaries and newly-attributed beneficiaries.
func (s *service) getNewAndExistingBeneficiaries(ctx context.Context, args worker_types.PrepareJobArgs) (newBeneficiaries, beneficiaries []*models.CCLFBeneficiary, err error) {
	cclfFileNew, err := s.repository.GetCCLFFileByID(ctx, args.CCLFFileNewID)
	if err != nil {
		return nil, nil, err
	}
	if cclfFileNew == nil {
		return nil, nil, fmt.Errorf("no CCLF8 file found for cmsID %s", args.CMSID)
	}

	if !args.Since.IsZero() && cclfFileNew.CreatedAt.Sub(args.Since) < 0 {
		// Retrieve all of the benes associated with this CCLF file.
		benes, err := s.getBenesByFileID(ctx, cclfFileNew.ID, args)
		if err != nil {
			return nil, nil, err
		}

		if len(benes) == 0 {
			return nil, nil, fmt.Errorf("found 0 new beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d", args.CMSID, cclfFileNew.ID)
		}

		return newBeneficiaries, benes, nil
	}

	cclfFileOld, err := s.repository.GetCCLFFileByID(ctx, args.CCLFFileOldID)
	if err != nil {
		return nil, nil, err
	}

	if cclfFileOld == nil {
		s.logger.Infof("Unable to find CCLF8 File for cmsID %s prior to date: %s; all beneficiaries will be considered NEW", args.CMSID, args.Since)

		newBeneficiaries, err = s.getBenesByFileID(ctx, cclfFileNew.ID, args)
		if err != nil {
			return nil, nil, err
		}
		if len(newBeneficiaries) == 0 {
			return nil, nil, fmt.Errorf("found 0 new beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d", args.CMSID, cclfFileNew.ID)
		}
		return newBeneficiaries, nil, nil
	}

	oldMBIs, err := s.repository.GetCCLFBeneficiaryMBIs(ctx, cclfFileOld.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve MBIs for cmsID %s cclfFileID %d %s", args.CMSID, cclfFileOld.ID, err.Error())
	}

	// Retrieve all of the benes associated with this CCLF file.
	benes, err := s.getBenesByFileID(ctx, cclfFileNew.ID, args)
	if err != nil {
		return nil, nil, err
	}
	if len(benes) == 0 {
		return nil, nil, fmt.Errorf("found 0 new or existing beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d", args.CMSID, cclfFileNew.ID)
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

func (s *service) getBeneficiaries(ctx context.Context, args worker_types.PrepareJobArgs) ([]*models.CCLFBeneficiary, error) {
	cclfFile, err := s.repository.GetCCLFFileByID(ctx, args.CCLFFileNewID)
	if err != nil {
		return nil, err
	}
	if cclfFile == nil {
		return nil, fmt.Errorf("no CCLF8 file found for cmsID %s", args.CMSID)
	}

	benes, err := s.getBenesByFileID(ctx, cclfFile.ID, args)
	if err != nil {
		return nil, err
	}
	if len(benes) == 0 {
		return nil, fmt.Errorf("found 0 beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d", args.CMSID, cclfFile.ID)
	}

	return benes, nil
}

func (s *service) getBenesByFileID(ctx context.Context, cclfFileID uint, args worker_types.PrepareJobArgs) ([]*models.CCLFBeneficiary, error) {
	var (
		ignoredMBIs []string
		err         error
	)
	if !s.sp.includeSuppressedBeneficiaries {
		upperBound := args.OptOutDate
		if args.OptOutDate.IsZero() {
			upperBound = time.Now()
		}

		if cfg, ok := s.GetACOConfigForID(args.CMSID); ok {
			if !cfg.IgnoreSuppressions {
				ignoredMBIs, err = s.repository.GetSuppressedMBIs(ctx, s.sp.lookbackDays, upperBound)
				if err != nil {
					return nil, fmt.Errorf("failed to retreive suppressedMBIs %s", err.Error())
				}
			}
		} else {
			return nil, fmt.Errorf("failed to load or match ACO config (or potentially no ACO Configs set), CMS ID: %+v", args.CMSID)
		}

	}

	benes, err := s.repository.GetCCLFBeneficiaries(ctx, cclfFileID, ignoredMBIs)
	if err != nil {
		return nil, fmt.Errorf("failed to get beneficiaries %s", err.Error())
	}

	return benes, nil
}

// setClaimsDate computes the claims window to apply on the args
func (s *service) setClaimsDate(args *worker_types.JobEnqueueArgs, prepareArgs worker_types.PrepareJobArgs) bool {
	// If the caller made a request for runout data
	// it takes precedence over any other claims date
	// that may be applied
	if prepareArgs.RequestType == constants.Runout {
		args.ClaimsWindow.UpperBound = s.rp.claimThruDate
	} else if !prepareArgs.ClaimsDate.IsZero() {
		args.ClaimsWindow.UpperBound = prepareArgs.ClaimsDate
	}

	// Applies the lower bound from the first matching ACOConfig
	cfg, ok := s.GetACOConfigForID(prepareArgs.CMSID)
	if ok {
		args.ClaimsWindow.LowerBound = cfg.LookbackTime()
	}

	return ok
}

// Gets the priority for the job where the lower the number the higher the priority in the queue.
// Priority is based on the request parameters that the job is executing on.
// Note: River queue library requires a priority between 1 and 4 (inclusive)
func (s *service) GetJobPriority(acoID string, resourceType string, sinceParam bool) int16 {
	var priority int16
	if isPriorityACO(acoID) {
		priority = int16(1) // priority level for jobs for synthetic ACOs that are used for smoke testing
	} else if resourceType == "Patient" || resourceType == "Coverage" {
		priority = int16(2) // priority level for jobs that only request smaller resources
	} else if sinceParam {
		priority = int16(3) // priority level for jobs that only request data for a limited timeframe
	} else {
		priority = int16(4) // default priority level for jobs
	}
	return priority
}

// GetACOConfigForID gets first matching currently loaded ACOConfig for the specified cmsID
func (s *service) GetACOConfigForID(cmsID string) (*ACOConfig, bool) {
	for _, cfg := range s.acoConfigs {
		if cfg.patternExp.MatchString(cmsID) {
			return &cfg, true
		}
	}

	return nil, false
}

// Checks to see if an ACO is priority ACO based on a regex pattern provided by an
// environment variable.
func isPriorityACO(acoID string) bool {
	if priorityACOPattern := conf.GetEnv("PRIORITY_ACO_REG_EX"); priorityACOPattern != "" {
		priorityACORegex := regexp.MustCompile(priorityACOPattern)
		if priorityACORegex.MatchString(acoID) {
			return true
		}
	}
	return false
}

func getMaxBeneCount(requestType string) (int, error) {
	const (
		BCDA_FHIR_MAX_RECORDS_EOB_DEFAULT           = 50
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
		mdtcoc  = `^CT\d{4,6}$`
		cdac    = `^DA\d{4}$`
		test    = `^TEST\d{3}$`
		sandbox = `^SBX[A-Z]{2}\d{3}$`
		pattern = `(` + ssp + `)|(` + ngaco + `)|(` + cec + `)|(` + ckcc + `)|(` + kcf + `)|(` + dc + `)|(` + mdtcoc + `)|(` + cdac + `)|(` + test + `)|(` + sandbox + `)`
	)

	return regexp.MustCompile(pattern).MatchString(cmsID)
}

func (s *service) GetLatestCCLFFile(ctx context.Context, cmsID string, lowerBound time.Time, upperBound time.Time, fileType models.CCLFFileType) (*models.CCLFFile, error) {
	cclfFile, err := s.repository.GetLatestCCLFFile(ctx, cmsID, constants.CCLF8FileNum, constants.ImportComplete, lowerBound, upperBound, fileType)
	if err != nil {
		return nil, err
	}

	if cclfFile == nil {
		return nil, CCLFNotFoundError{8, cmsID, fileType, time.Time{}}
	}

	return cclfFile, nil
}

// GetTimeConstraints searches for any time bounds that we should apply on the associated ACO
func (s *service) GetTimeConstraints(ctx context.Context, cmsID string) (TimeConstraints, error) {
	var constraint TimeConstraints
	aco, err := s.repository.GetACOByCMSID(ctx, cmsID)
	if err != nil {
		return constraint, fmt.Errorf("failed to retrieve aco: %w", err)
	}

	// If aco is not terminated, then we should not apply any time constraints
	if aco.TerminationDetails == nil {
		return constraint, nil
	}

	constraint.AttributionDate = aco.TerminationDetails.AttributionDate()
	constraint.ClaimsDate = aco.TerminationDetails.ClaimsDate()
	constraint.OptOutDate = aco.TerminationDetails.OptOutDate()
	return constraint, nil
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
	ErrJobNotCancelled   = goerrors.New("job was not cancelled due to internal server error")
	ErrJobNotCancellable = goerrors.New("job was not cancelled because it is not Pending or In Progress")
)
