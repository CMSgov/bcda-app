package models

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bgentry/que-go"
	"github.com/pkg/errors"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/utils"
	log "github.com/sirupsen/logrus"
)

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
	GetQueJobs(ctx context.Context, cmsID string, job *Job, resourceTypes []string, since time.Time, reqType RequestType) (queJobs []*que.Job, err error)

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

func (s *service) GetQueJobs(ctx context.Context, cmsID string, job *Job, resourceTypes []string, since time.Time, reqType RequestType) (queJobs []*que.Job, err error) {
	fileType := FileTypeDefault
	switch reqType {
	case Runout:
		fileType = FileTypeRunout
		fallthrough
	case DefaultRequest:
		beneficiaries, err := s.getBeneficiaries(ctx, cmsID, fileType)
		if err != nil {
			return nil, err
		}

		// add beneficaries to the job queue
		jobs, err := s.createQueueJobs(job, cmsID, resourceTypes, since, beneficiaries, reqType)
		if err != nil {
			return nil, err
		}
		queJobs = append(queJobs, jobs...)
	case RetrieveNewBeneHistData:
		newBeneficiaries, beneficiaries, err := s.getNewAndExistingBeneficiaries(ctx, cmsID, since)
		if err != nil {
			return nil, err
		}

		// add new beneficaries to the job queue
		jobs, err := s.createQueueJobs(job, cmsID, resourceTypes, time.Time{}, newBeneficiaries, reqType)
		if err != nil {
			return nil, err
		}
		queJobs = append(queJobs, jobs...)

		// add existing beneficaries to the job queue
		jobs, err = s.createQueueJobs(job, cmsID, resourceTypes, since, beneficiaries, reqType)
		if err != nil {
			return nil, err
		}
		queJobs = append(queJobs, jobs...)
	default:
		return nil, fmt.Errorf("Unsupported RequestType %d", reqType)
	}

	return queJobs, nil
}

func (s *service) GetJobAndKeys(ctx context.Context, jobID uint) (*Job, []*JobKey, error) {
	j, err := s.repository.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, nil, err
	}

	// No need to look up job keys if the
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
	// Get the job by id.
	job, err := s.repository.GetJobByID(ctx, jobID)
	if err != nil {
		return 0, err
	}

	// Check if the job is pending or in progress.
	if job.Status == JobStatusPending || job.Status == JobStatusInProgress {
		job.Status = JobStatusCancelled
		err = s.repository.UpdateJob(ctx, *job)
		if err != nil {
			return 0, err
		}
		return jobID, nil
	}

	// Return 0, nil to indicate attempt to cancel a non-cancellable job.
	return 0, nil
}

func (s *service) createQueueJobs(job *Job, CMSID string, resourceTypes []string, since time.Time, beneficiaries []*CCLFBeneficiary, reqType RequestType) (jobs []*que.Job, err error) {

	// persist in format ready for usage with _lastUpdated -- i.e., prepended with 'gt'
	var sinceArg string
	if !since.IsZero() {
		sinceArg = "gt" + since.Format(time.RFC3339Nano)
	}

	for _, rt := range resourceTypes {
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
					ID:              int(job.ID),
					ACOID:           job.ACOID.String(),
					BeneficiaryIDs:  jobIDs,
					ResourceType:    rt,
					Since:           sinceArg,
					TransactionTime: job.TransactionTime,
					BBBasePath:      s.bbBasePath,
				}

				if reqType == Runout {
					enqueueArgs.ServiceDate = s.rp.claimThruDate
				}

				args, err := json.Marshal(enqueueArgs)
				if err != nil {
					return nil, err
				}

				j := &que.Job{
					Type:     "ProcessJob",
					Args:     args,
					Priority: getJobPriority(CMSID, rt, (!since.IsZero() || reqType == RetrieveNewBeneHistData)),
				}

				jobs = append(jobs, j)
				jobIDs = make([]string, 0, maxBeneficiaries)
			}
		}
	}

	return jobs, nil
}

func (s *service) getNewAndExistingBeneficiaries(ctx context.Context, cmsID string, since time.Time) (newBeneficiaries, beneficiaries []*CCLFBeneficiary, err error) {

	var (
		cutoffTime time.Time
	)

	if s.stdCutoffDuration > 0 {
		cutoffTime = time.Now().Add(-1 * s.stdCutoffDuration)
	}

	cclfFileNew, err := s.repository.GetLatestCCLFFile(ctx, cmsID, cclf8FileNum, constants.ImportComplete, cutoffTime, time.Time{}, FileTypeDefault)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get new CCLF file for cmsID %s %s", cmsID, err.Error())
	}
	if cclfFileNew == nil {
		return nil, nil, CCLFNotFoundError{8, cmsID, FileTypeDefault, cutoffTime}
	}

	cclfFileOld, err := s.repository.GetLatestCCLFFile(ctx, cmsID, cclf8FileNum, constants.ImportComplete, time.Time{}, since, FileTypeDefault)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get old CCLF file for cmsID %s %s", cmsID, err.Error())
	}

	if cclfFileOld == nil {
		s.logger.Infof("Unable to find CCLF8 File for cmsID %s prior to date: %s; all beneficiaries will be considered NEW", cmsID, since)
		newBeneficiaries, err = s.getBenesByFileID(ctx, cclfFileNew.ID)
		if err != nil {
			return nil, nil, err
		}
		if len(newBeneficiaries) == 0 {
			return nil, nil, fmt.Errorf("Found 0 new beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d", cmsID, cclfFileNew.ID)
		}
		return newBeneficiaries, nil, nil
	}

	oldMBIs, err := s.repository.GetCCLFBeneficiaryMBIs(ctx, cclfFileOld.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve MBIs for cmsID %s cclfFileID %d %s", cmsID, cclfFileOld.ID, err.Error())
	}

	// Retrieve all of the benes associated with this CCLF file.
	benes, err := s.getBenesByFileID(ctx, cclfFileNew.ID)
	if err != nil {
		return nil, nil, err
	}
	if len(benes) == 0 {
		return nil, nil, fmt.Errorf("Found 0 new or existing beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d", cmsID, cclfFileNew.ID)
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

func (s *service) getBeneficiaries(ctx context.Context, cmsID string, fileType CCLFFileType) ([]*CCLFBeneficiary, error) {
	var (
		cutoffTime time.Time
	)

	if fileType == FileTypeDefault && s.stdCutoffDuration > 0 {
		cutoffTime = time.Now().Add(-1 * s.stdCutoffDuration)
	} else if fileType == FileTypeRunout && s.rp.cutoffDuration > 0 {
		cutoffTime = time.Now().Add(-1 * s.rp.cutoffDuration)
	}

	cclfFile, err := s.repository.GetLatestCCLFFile(ctx, cmsID, cclf8FileNum, constants.ImportComplete, cutoffTime, time.Time{}, fileType)
	if err != nil {
		return nil, fmt.Errorf("failed to get CCLF file for cmsID %s fileType %d %s",
			cmsID, fileType, err.Error())
	}
	if cclfFile == nil {
		return nil, CCLFNotFoundError{8, cmsID, fileType, cutoffTime}
	}

	benes, err := s.getBenesByFileID(ctx, cclfFile.ID)
	if err != nil {
		return nil, err
	}
	if len(benes) == 0 {
		return nil, fmt.Errorf("Found 0 beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d",
			cmsID, cclfFile.ID)
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
	if priorityACOPattern := os.Getenv("PRIORITY_ACO_REG_EX"); priorityACOPattern != "" {
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
		pattern = `(` + ssp + `)|(` + ngaco + `)|(` + cec + `)`
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
