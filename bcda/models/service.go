package models

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bgentry/que-go"

	"github.com/CMSgov/bcda-app/bcda/constants"
	log "github.com/sirupsen/logrus"
)

// Ensure service satisfies the interface
var _ Service = &service{}

// Service contains all of the methods needed to interact with the data represented in the models package
type Service interface {
	cclfBeneficiaryService
}

type cclfBeneficiaryService interface {
	GetQueJobs(cmsID string, job *Job, resourceTypes []string, since string, reqType RequestType) (queJobs []*que.Job, err error)
}

const (
	cclf8FileNum = int(8)
)

func NewService(r Repository, cutoffDuration time.Duration, lookbackDays int) Service {
	return &service{
		repository:     r,
		logger:         log.StandardLogger(),
		cutoffDuration: cutoffDuration,
		sp: suppressionParameters{
			includeSuppressedBeneficiaries: false,
			lookbackDays:                   lookbackDays,
		},
	}
}

type service struct {
	repository Repository

	logger *log.Logger

	cutoffDuration time.Duration
	sp             suppressionParameters
}

type suppressionParameters struct {
	includeSuppressedBeneficiaries bool
	lookbackDays                   int
}

func (s *service) GetQueJobs(cmsID string, job *Job, resourceTypes []string, since string, reqType RequestType) (queJobs []*que.Job, err error) {
	fileType := FileTypeDefault
	switch reqType {
	case Runout:
		fileType = FileTypeRunout
		fallthrough
	case DefaultRequest:
		beneficiaries, err := s.getBeneficiaries(cmsID, fileType)
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
		t, err := time.Parse(time.RFC3339Nano, since)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s using format %s", since, time.RFC3339Nano)
		}

		newBeneficiaries, beneficiaries, err := s.getNewAndExistingBeneficiaries(cmsID, t)
		if err != nil {
			return nil, err
		}

		// add new beneficaries to the job queue
		jobs, err := s.createQueueJobs(job, cmsID, resourceTypes, "", newBeneficiaries, reqType)
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

func (s *service) createQueueJobs(job *Job, CMSID string, resourceTypes []string, since string, beneficiaries []*CCLFBeneficiary, reqType RequestType) (jobs []*que.Job, err error) {

	// persist in format ready for usage with _lastUpdated -- i.e., prepended with 'gt'
	if since != "" {
		since = "gt" + since
	}
	for _, rt := range resourceTypes {
		maxBeneficiaries, err := GetMaxBeneCount(rt)
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
					Since:           since,
					TransactionTime: job.TransactionTime,
				}

				if reqType == Runout {
					enqueueArgs.ServiceDate = time.Time{} // TODO - FILL ME IN
				}

				args, err := json.Marshal(enqueueArgs)
				if err != nil {
					return nil, err
				}

				j := &que.Job{
					Type:     "ProcessJob",
					Args:     args,
					Priority: setJobPriority(CMSID, rt, (len(since) != 0 || reqType == RetrieveNewBeneHistData)),
				}

				jobs = append(jobs, j)
				jobIDs = make([]string, 0, maxBeneficiaries)
			}
		}
	}

	return jobs, nil
}

func (s *service) getNewAndExistingBeneficiaries(cmsID string, since time.Time) (newBeneficiaries, beneficiaries []*CCLFBeneficiary, err error) {

	var (
		cutoffTime time.Time
	)

	if s.cutoffDuration > 0 {
		cutoffTime = time.Now().Add(-1 * s.cutoffDuration)
	}

	cclfFileNew, err := s.repository.GetLatestCCLFFile(cmsID, cclf8FileNum, constants.ImportComplete, cutoffTime, time.Time{}, FileTypeDefault)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get new CCLF file for cmsID %s %s", cmsID, err.Error())
	}
	if cclfFileNew == nil {
		return nil, nil, fmt.Errorf("no CCLF8 file found for cmsID %s cutoffTime %s", cmsID, cutoffTime.String())
	}

	cclfFileOld, err := s.repository.GetLatestCCLFFile(cmsID, cclf8FileNum, constants.ImportComplete, time.Time{}, since, FileTypeDefault)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get old CCLF file for cmsID %s %s", cmsID, err.Error())
	}

	if cclfFileOld == nil {
		s.logger.Infof("Unable to find CCLF8 File for cmsID %s prior to date: %s; all beneficiaries will be considered NEW", cmsID, since)
		newBeneficiaries, err = s.getBenesByFileID(cclfFileNew.ID)
		if err != nil {
			return nil, nil, err
		}
		if len(newBeneficiaries) == 0 {
			return nil, nil, fmt.Errorf("Found 0 new beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d", cmsID, cclfFileNew.ID)
		}
		return newBeneficiaries, nil, nil
	}

	oldMBIs, err := s.repository.GetCCLFBeneficiaryMBIs(cclfFileOld.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retreive MBIs for cmsID %s cclfFileID %d %s", cmsID, cclfFileOld.ID, err.Error())
	}

	// Retrieve all of the benes associated with this CCLF file.
	benes, err := s.getBenesByFileID(cclfFileNew.ID)
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

func (s *service) getBeneficiaries(cmsID string, fileType CCLFFileType) ([]*CCLFBeneficiary, error) {
	var (
		cutoffTime time.Time
	)

	if s.cutoffDuration > 0 {
		cutoffTime = time.Now().Add(-1 * s.cutoffDuration)
	}

	cclfFile, err := s.repository.GetLatestCCLFFile(cmsID, cclf8FileNum, constants.ImportComplete, cutoffTime, time.Time{}, fileType)
	if err != nil {
		return nil, fmt.Errorf("failed to get CCLF file for cmsID %s %s", cmsID, err.Error())
	}
	if cclfFile == nil {
		return nil, fmt.Errorf("no CCLF8 file found for cmsID %s cutoffTime %s", cmsID, cutoffTime.String())
	}

	benes, err := s.getBenesByFileID(cclfFile.ID)
	if err != nil {
		return nil, err
	}
	if len(benes) == 0 {
		return nil, fmt.Errorf("Found 0 beneficiaries from CCLF8 file for cmsID %s cclfFiledID %d", cmsID, cclfFile.ID)
	}

	return benes, nil
}

func (s *service) getBenesByFileID(cclfFileID uint) ([]*CCLFBeneficiary, error) {
	var (
		ignoredMBIs []string
		err         error
	)
	if !s.sp.includeSuppressedBeneficiaries {
		ignoredMBIs, err = s.repository.GetSuppressedMBIs(s.sp.lookbackDays)
		if err != nil {
			return nil, fmt.Errorf("failed to retreive suppressedMBIs %s", err.Error())
		}
	}

	benes, err := s.repository.GetCCLFBeneficiaries(cclfFileID, ignoredMBIs)
	if err != nil {
		return nil, fmt.Errorf("failed to get beneficiaries %s", err.Error())
	}

	return benes, nil
}

// Sets the priority for the job where the lower the number the higher the priority in the queue.
// Prioirity is based on the request parameters that the job is executing on.
func setJobPriority(acoID string, resourceType string, sinceParam bool) int16 {
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
