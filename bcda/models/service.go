package models

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bgentry/que-go"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"
	log "github.com/sirupsen/logrus"
)

type RequestConditions struct {
	ReqType   RequestType
	Resources []string

	CMSID string
	ACOID uuid.UUID

	JobID           uint
	Since           time.Time
	TransactionTime time.Time

	// Fields set in the service
	fileType CCLFFileType

	attributionDate time.Time
	optOptDate      time.Time
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
	GetQueJobs(ctx context.Context, conditions RequestConditions) (queJobs []*que.Job, err error)

	GetJobAndKeys(ctx context.Context, jobID uint) (*Job, []*JobKey, error)

	CancelJob(ctx context.Context, jobID uint) (uint, error)
}

const (
	cclf8FileNum = int(8)
)

func NewService(r Repository, cutoffDuration time.Duration, lookbackDays int,
	runoutCutoffDuration time.Duration, runoutClaimThru time.Time,
	basePath string) Service {
	return &service{
		repository:        r,
		logger:            log.StandardLogger(),
		stdCutoffDuration: cutoffDuration,
		sp: suppressionParameters{
			includeSuppressedBeneficiaries: false,
			lookbackDays:                   lookbackDays,
		},
		rp: runoutParameters{
			// Runouts apply to claims data for the previous year.
			claimThruDate:  runoutClaimThru,
			cutoffDuration: runoutCutoffDuration,
		},
		bbBasePath: basePath,
	}
}

type service struct {
	repository Repository

	logger *log.Logger

	stdCutoffDuration time.Duration
	sp                suppressionParameters
	rp                runoutParameters
	bbBasePath        string
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

func (s *service) GetQueJobs(ctx context.Context, conditions RequestConditions) (queJobs []*que.Job, err error) {

	if err := s.setTimeConstraints(ctx, conditions.ACOID, &conditions); err != nil {
		return nil, fmt.Errorf("failed to set time constraints for caller: %w", err)
	}

	conditions.fileType = FileTypeDefault
	switch conditions.ReqType {
	case Runout:
		conditions.fileType = FileTypeRunout
		fallthrough
	case DefaultRequest:
		beneficiaries, err := s.getBeneficiaries(ctx, conditions)
		if err != nil {
			return nil, err
		}

		// add beneficaries to the job queue
		jobs, err := s.createQueueJobs(conditions, conditions.Since, beneficiaries)
		if err != nil {
			return nil, err
		}
		queJobs = append(queJobs, jobs...)
	case RetrieveNewBeneHistData:
		newBeneficiaries, beneficiaries, err := s.getNewAndExistingBeneficiaries(ctx, conditions)
		if err != nil {
			return nil, err
		}

		// add new beneficaries to the job queue use a default time value to ensure
		// that we retrieve the full history for these beneficiaries
		jobs, err := s.createQueueJobs(conditions, time.Time{}, newBeneficiaries)
		if err != nil {
			return nil, err
		}
		queJobs = append(queJobs, jobs...)

		// add existing beneficaries to the job queue
		jobs, err = s.createQueueJobs(conditions, conditions.Since, beneficiaries)
		if err != nil {
			return nil, err
		}
		queJobs = append(queJobs, jobs...)
	default:
		return nil, fmt.Errorf("Unsupported RequestType %d", conditions.ReqType)
	}

	return queJobs, nil
}

func (s *service) GetJobAndKeys(ctx context.Context, jobID uint) (*Job, []*JobKey, error) {
	j, err := s.repository.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, nil, err
	}

	// No need to look up job keys if the job is complete
	if j.Status != JobStatusCompleted {
		return j, nil, nil
	}

	keys, err := s.repository.GetJobKeys(ctx, jobID)
	if err != nil {
		return nil, nil, err
	}

	nonEmptyKeys := make([]*JobKey, 0, len(keys))
	for i, key := range keys {
		if strings.TrimSpace(key.FileName) == BlankFileName {
			continue
		}
		nonEmptyKeys = append(nonEmptyKeys, keys[i])
	}

	return j, nonEmptyKeys, nil
}

func (s *service) CancelJob(ctx context.Context, jobID uint) (uint, error) {
	// Assumes the job exists and retrieves the job by ID
	job, err := s.repository.GetJobByID(ctx, jobID)
	if err != nil {
		return 0, err
	}

	// Check if the job is pending or in progress.
	if job.Status == JobStatusPending || job.Status == JobStatusInProgress {
		job.Status = JobStatusCancelled
		err = s.repository.UpdateJob(ctx, *job)
		if err != nil {
			return 0, ErrJobNotCancelled
		}
		return jobID, nil
	}

	// Return 0, ErrJobNotCancellable when attempting to cancel a non-cancellable job.
	return 0, ErrJobNotCancellable
}

func (s *service) createQueueJobs(conditions RequestConditions, since time.Time, beneficiaries []*CCLFBeneficiary) (jobs []*que.Job, err error) {

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
				enqueueArgs := JobEnqueueArgs{
					ID:              int(conditions.JobID),
					ACOID:           conditions.ACOID.String(),
					BeneficiaryIDs:  jobIDs,
					ResourceType:    rt,
					Since:           sinceArg,
					TransactionTime: conditions.TransactionTime,
					BBBasePath:      s.bbBasePath,
				}

				if conditions.ReqType == Runout {
					enqueueArgs.ServiceDate = s.rp.claimThruDate
				}

				args, err := json.Marshal(enqueueArgs)
				if err != nil {
					return nil, err
				}

				j := &que.Job{
					Type:     "ProcessJob",
					Args:     args,
					Priority: getJobPriority(conditions.CMSID, rt, (!since.IsZero() || conditions.ReqType == RetrieveNewBeneHistData)),
				}

				jobs = append(jobs, j)
				jobIDs = make([]string, 0, maxBeneficiaries)
			}
		}
	}

	return jobs, nil
}

func (s *service) getNewAndExistingBeneficiaries(ctx context.Context, conditions RequestConditions) (newBeneficiaries, beneficiaries []*CCLFBeneficiary, err error) {

	var (
		cutoffTime time.Time
	)

	if s.stdCutoffDuration > 0 {
		cutoffTime = time.Now().Add(-1 * s.stdCutoffDuration)
	}

	cclfFileNew, err := s.repository.GetLatestCCLFFile(ctx, conditions.CMSID, cclf8FileNum, constants.ImportComplete,
		cutoffTime, time.Time{}, conditions.fileType)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get new CCLF file for cmsID %s %s", conditions.CMSID, err.Error())
	}
	if cclfFileNew == nil {
		return nil, nil, CCLFNotFoundError{8, conditions.CMSID, conditions.fileType, cutoffTime}
	}

	cclfFileOld, err := s.repository.GetLatestCCLFFile(ctx, conditions.CMSID, cclf8FileNum, constants.ImportComplete,
		time.Time{}, conditions.Since, FileTypeDefault)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get old CCLF file for cmsID %s %s", conditions.CMSID, err.Error())
	}

	if cclfFileOld == nil {
		s.logger.Infof("Unable to find CCLF8 File for cmsID %s prior to date: %s; all beneficiaries will be considered NEW",
			conditions.CMSID, conditions.Since)
		newBeneficiaries, err = s.getBenesByFileID(ctx, cclfFileNew.ID)
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
	benes, err := s.getBenesByFileID(ctx, cclfFileNew.ID)
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

func (s *service) getBeneficiaries(ctx context.Context, conditions RequestConditions) ([]*CCLFBeneficiary, error) {
	var (
		cutoffTime time.Time
	)

	if conditions.fileType == FileTypeDefault && s.stdCutoffDuration > 0 {
		cutoffTime = time.Now().Add(-1 * s.stdCutoffDuration)
	} else if conditions.fileType == FileTypeRunout && s.rp.cutoffDuration > 0 {
		cutoffTime = time.Now().Add(-1 * s.rp.cutoffDuration)
	}

	cclfFile, err := s.repository.GetLatestCCLFFile(ctx, conditions.CMSID, cclf8FileNum,
		constants.ImportComplete, cutoffTime, time.Time{}, conditions.fileType)
	if err != nil {
		return nil, fmt.Errorf("failed to get CCLF file for cmsID %s fileType %d %s",
			conditions.CMSID, conditions.fileType, err.Error())
	}
	if cclfFile == nil {
		return nil, CCLFNotFoundError{8, conditions.CMSID, conditions.fileType, cutoffTime}
	}

	benes, err := s.getBenesByFileID(ctx, cclfFile.ID)
	if err != nil {
		return nil, err
	}
	if len(benes) == 0 {
		return nil, fmt.Errorf("Found 0 beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d",
			conditions.CMSID, cclfFile.ID)
	}

	return benes, nil
}

func (s *service) getBenesByFileID(ctx context.Context, cclfFileID uint) ([]*CCLFBeneficiary, error) {
	var (
		ignoredMBIs []string
		err         error
	)
	if !s.sp.includeSuppressedBeneficiaries {
		ignoredMBIs, err = s.repository.GetSuppressedMBIs(ctx, s.sp.lookbackDays)
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

// setTimeConstraints searches for any time bounds that we should apply on the associated ACO
func (s *service) setTimeConstraints(ctx context.Context, acoID uuid.UUID, conditions *RequestConditions) error {
	aco, err := s.repository.GetACOByUUID(ctx, acoID)
	if err != nil {
		return fmt.Errorf("failed to retrieve aco: %w", err)
	}

	// If aco is not terminated, then we should not apply any time constraints
	if aco.TerminationDetails == nil {
		conditions.attributionDate = time.Time{}
		conditions.claimsDate = time.Time{}
		conditions.optOptDate = time.Time{}
		return nil
	}

	conditions.attributionDate = aco.TerminationDetails.AttributionDate()
	conditions.claimsDate = aco.TerminationDetails.ClaimsDate()
	conditions.optOptDate = aco.TerminationDetails.OptOutDate()
	return nil
}

// Gets the priority for the job where the lower the number the higher the priority in the queue.
// Priority is based on the request parameters that the job is executing on.
func getJobPriority(acoID string, resourceType string, sinceParam bool) int16 {
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
		BCDA_FHIR_MAX_RECORDS_EOB_DEFAULT      = 200
		BCDA_FHIR_MAX_RECORDS_PATIENT_DEFAULT  = 5000
		BCDA_FHIR_MAX_RECORDS_COVERAGE_DEFAULT = 4000
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
		pattern = `(` + ssp + `)|(` + ngaco + `)|(` + cec + `)|(` + ckcc + `)|(` + kcf + `)|(` + dc + `)`
	)

	return regexp.MustCompile(pattern).MatchString(cmsID)
}

type CCLFNotFoundError struct {
	FileNumber int
	CMSID      string
	FileType   CCLFFileType
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
