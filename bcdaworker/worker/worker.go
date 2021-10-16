package worker

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/CMSgov/bcda-app/bcda/cclf/metrics"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	fhirmodels "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/sirupsen/logrus"

	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

type Worker interface {
	ValidateJob(ctx context.Context, jobArgs models.JobEnqueueArgs) (*models.Job, error)
	ProcessJob(ctx context.Context, job models.Job, jobArgs models.JobEnqueueArgs) error
}

type worker struct {
	r repository.Repository
}

func NewWorker(db *sql.DB) Worker {
	return &worker{postgres.NewRepository(db)}
}

func (w *worker) ValidateJob(ctx context.Context, jobArgs models.JobEnqueueArgs) (*models.Job, error) {
	if len(jobArgs.BBBasePath) == 0 {
		return nil, ErrNoBasePathSet
	}

	exportJob, err := w.r.GetJobByID(ctx, uint(jobArgs.ID))
	if goerrors.Is(err, repository.ErrJobNotFound) {
		return nil, ErrParentJobNotFound
	} else if err != nil {
		return nil, errors.Wrap(err, "could not retrieve job from database")
	}

	if exportJob.Status == models.JobStatusCancelled {
		return nil, ErrParentJobCancelled
	}

	return exportJob, nil
}

func (w *worker) ProcessJob(ctx context.Context, job models.Job, jobArgs models.JobEnqueueArgs) error {

	t := metrics.GetTimer()
	defer t.Close()
	ctx = metrics.NewContext(ctx, t)
	ctx, c := metrics.NewParent(ctx, fmt.Sprintf("ProcessJob-%s", jobArgs.ResourceType))
	defer c()

	aco, err := w.r.GetACOByUUID(ctx, job.ACOID)
	if err != nil {
		return errors.Wrap(err, "could not retrieve ACO from database")
	}

	err = w.r.UpdateJobStatusCheckStatus(ctx, job.ID, models.JobStatusPending, models.JobStatusInProgress)
	if goerrors.Is(err, repository.ErrJobNotUpdated) {
		// could also occur if job was marked as cancelled
		log.Worker.Warnf("Failed to update job. Assume job already updated. Continuing. %s", err.Error())
	} else if err != nil {
		return errors.Wrap(err, "could not update job status in database")
	}

	bb, err := client.NewBlueButtonClient(client.NewConfig(jobArgs.BBBasePath))
	if err != nil {
		err = errors.Wrap(err, "could not create Blue Button client")
		log.Worker.Error(err)
		return err
	}

	jobID := strconv.Itoa(jobArgs.ID)
	stagingPath := fmt.Sprintf("%s/%s", conf.GetEnv("FHIR_STAGING_DIR"), jobID)
	payloadPath := fmt.Sprintf("%s/%s", conf.GetEnv("FHIR_PAYLOAD_DIR"), jobID)

	if err = createDir(stagingPath); err != nil {
		log.Worker.Error(err)
		return err
	}

	// Create directory for job results.
	// This will be used in the clean up later to move over processed files.
	if err = createDir(payloadPath); err != nil {
		log.Worker.Error(err)
		return err
	}

	fileUUID, fileSize, err := writeBBDataToFile(ctx, w.r, bb, *aco.CMSID, jobArgs)
	fileName := fileUUID + ".ndjson"

	// This is only run AFTER completion of all the collection
	if err != nil {
		// only inProgress jobs should move to a failed status (i.e. don't move a cancelled job to failed)
		err = w.r.UpdateJobStatusCheckStatus(ctx, job.ID, models.JobStatusInProgress, models.JobStatusFailed)
		if goerrors.Is(err, repository.ErrJobNotUpdated) {
			log.Worker.Warnf("Failed to update job (job cancelled): %s", err)
			return err
		} else if err != nil {
			return err
		}
	} else {
		if fileSize == 0 {
			log.Worker.Warn("Empty file found in request: ", fileName)
			fileName = models.BlankFileName
		}

		jk := models.JobKey{JobID: job.ID, FileName: fileName, ResourceType: jobArgs.ResourceType}
		if err := w.r.CreateJobKey(ctx, jk); err != nil {
			log.Worker.Error(err)
			return err
		}
	}

	_, err = checkJobCompleteAndCleanup(ctx, w.r, job.ID)
	if err != nil {
		log.Worker.Error(err)
		return err
	}

	// Not critical since we use the job_keys count as the authoritative list of completed jobs.
	// CompletedJobCount is purely information and can be off.
	if err := w.r.IncrementCompletedJobCount(ctx, job.ID); err != nil {
		log.Worker.Warnf("Failed to update completed job count for job %d. Will continue. %s", job.ID, err.Error())
	}

	return nil
}

func writeBBDataToFile(ctx context.Context, r repository.Repository, bb client.APIClient,
	cmsID string, jobArgs models.JobEnqueueArgs) (fileUUID string, size int64, err error) {

	close := metrics.NewChild(ctx, "writeBBDataToFile")
	defer close()

	var bundleFunc func(bene models.CCLFBeneficiary) (*fhirmodels.Bundle, error)
	// NOTE: Currently all Coverage/EOB/Patient requests are for adjudicated data and
	// Claim/ClaimResponse are pre-adjudicated, future work may require checking what
	// kind of backing data to pull from if there is overlap (one or more FHIR resource
	// used for representing both adjudicated and pre-adjudicated data)
	switch jobArgs.ResourceType {
	case "Coverage":
		bundleFunc = func(bene models.CCLFBeneficiary) (*fhirmodels.Bundle, error) {
			return bb.GetCoverage(bene.BlueButtonID, strconv.Itoa(jobArgs.ID), cmsID, jobArgs.Since, jobArgs.TransactionTime)
		}
	case "ExplanationOfBenefit":
		bundleFunc = func(bene models.CCLFBeneficiary) (*fhirmodels.Bundle, error) {
			cw := client.ClaimsWindow{
				LowerBound: jobArgs.ClaimsWindow.LowerBound,
				UpperBound: jobArgs.ClaimsWindow.UpperBound}
			return bb.GetExplanationOfBenefit(bene.BlueButtonID, strconv.Itoa(jobArgs.ID), cmsID, jobArgs.Since, jobArgs.TransactionTime,
				cw)
		}
	case "Patient":
		bundleFunc = func(bene models.CCLFBeneficiary) (*fhirmodels.Bundle, error) {
			return bb.GetPatient(bene.BlueButtonID, strconv.Itoa(jobArgs.ID), cmsID, jobArgs.Since, jobArgs.TransactionTime)
		}
		//NOTE: The assumption is Claim/ClaimResponse is always pre-adjudicated, future work may require checking what
		//kind of backing data to pull from
	case "Claim":
		bundleFunc = func(bene models.CCLFBeneficiary) (*fhirmodels.Bundle, error) {
			cw := client.ClaimsWindow{
				LowerBound: jobArgs.ClaimsWindow.LowerBound,
				UpperBound: jobArgs.ClaimsWindow.UpperBound}
			return bb.GetClaim(bene.MBI, strconv.Itoa(jobArgs.ID), cmsID, jobArgs.Since, jobArgs.TransactionTime, cw)
		}
	case "ClaimResponse":
		bundleFunc = func(bene models.CCLFBeneficiary) (*fhirmodels.Bundle, error) {
			cw := client.ClaimsWindow{
				LowerBound: jobArgs.ClaimsWindow.LowerBound,
				UpperBound: jobArgs.ClaimsWindow.UpperBound}
			return bb.GetClaimResponse(bene.MBI, strconv.Itoa(jobArgs.ID), cmsID, jobArgs.Since, jobArgs.TransactionTime, cw)
		}
	default:
		return "", 0, fmt.Errorf("unsupported resource type %s", jobArgs.ResourceType)
	}

	dataDir := conf.GetEnv("FHIR_STAGING_DIR")
	fileUUID = uuid.New()
	f, err := os.Create(fmt.Sprintf("%s/%d/%s.ndjson", dataDir, jobArgs.ID, fileUUID))
	if err != nil {
		log.Worker.Error(err)
		return "", 0, err
	}

	defer utils.CloseFileAndLogError(f)

	w := bufio.NewWriter(f)
	errorCount := 0
	totalBeneIDs := float64(len(jobArgs.BeneficiaryIDs))
	failThreshold := getFailureThreshold()
	failed := false

	for _, beneID := range jobArgs.BeneficiaryIDs {
		// if the parent job was cancelled, stop processing beneIDs and fail the job
		if ctx.Err() == context.Canceled {
			failed = true
			break
		}

		errMsg, code, err := func() (string, fhircodes.IssueTypeCode_Value, error) {
			id, err := strconv.ParseUint(beneID, 10, 64)
			if err != nil {
				return fmt.Sprintf("Error failed to convert %s to uint", beneID), fhircodes.IssueTypeCode_EXCEPTION, err
			}

			// NOTE: with adjudicated data sets, we first need to lookup the Patient ID
			// before gathering EOB/Coverage results; however with pre-adjudicated data
			// that is not yet possible because their are no Patient FHIR resources. This
			// boolean indicates whether or not we need to skip that lookup step
			fetchBBId := jobArgs.DataType != constants.PreAdjudicated
			bene, err := getBeneficiary(ctx, r, uint(id), bb, fetchBBId)
			if err != nil {
				return fmt.Sprintf("Error retrieving BlueButton ID for cclfBeneficiary MBI %s", bene.MBI), fhircodes.IssueTypeCode_NOT_FOUND, err
			}

			b, err := bundleFunc(bene)
			if err != nil {
				return fmt.Sprintf("Error retrieving %s for beneficiary MBI %s in ACO %s", jobArgs.ResourceType, bene.MBI, jobArgs.ACOID), fhircodes.IssueTypeCode_NOT_FOUND, err
			}
			fhirBundleToResourceNDJSON(ctx, w, b, jobArgs.ResourceType, beneID, cmsID, fileUUID, jobArgs.ID)
			return "", 0, nil
		}()

		if err != nil {
			log.Worker.WithFields(logrus.Fields{
				"jobID": jobArgs.ID,
				"cmsID": cmsID,
			}).Error(err)
			errorCount++
			appendErrorToFile(ctx, fileUUID, code, responseutils.BbErr, errMsg, jobArgs.ID)
		}

		failPct := (float64(errorCount) / totalBeneIDs) * 100
		if failPct >= failThreshold {
			failed = true
			break
		}
	}

	if err = w.Flush(); err != nil {
		return "", 0, err
	}

	if failed {
		if ctx.Err() == context.Canceled {
			return "", 0, errors.New(fmt.Sprintf("Parent job %d was cancelled", jobArgs.ID))
		}
		return "", 0, errors.New("number of failed requests has exceeded threshold")
	}

	fstat, err := f.Stat()
	if err != nil {
		return "", 0, err
	}

	return fileUUID, fstat.Size(), nil
}

// getBeneficiary returns the beneficiary. The bb ID value is retrieved and set in the model.
func getBeneficiary(ctx context.Context, r repository.Repository, beneID uint, bb client.APIClient, fetchBBId bool) (models.CCLFBeneficiary, error) {
	bene, err := r.GetCCLFBeneficiaryByID(ctx, beneID)
	if err != nil {
		return models.CCLFBeneficiary{}, err
	}

	cclfBeneficiary := *bene

	if fetchBBId {
		bbID, err := getBlueButtonID(bb, cclfBeneficiary.MBI)

		if err != nil {
			err = fmt.Errorf("failed to get blue button id for ID %d: %w", beneID, err)
			return cclfBeneficiary, err
		}

		cclfBeneficiary.BlueButtonID = bbID
	}

	return cclfBeneficiary, nil
}

func getFailureThreshold() float64 {
	exportFailPctStr := conf.GetEnv("EXPORT_FAIL_PCT")
	exportFailPct, err := strconv.Atoi(exportFailPctStr)
	if err != nil {
		exportFailPct = 50
	} else if exportFailPct < 0 {
		exportFailPct = 0
	} else if exportFailPct > 100 {
		exportFailPct = 100
	}
	return float64(exportFailPct)
}

func appendErrorToFile(ctx context.Context, fileUUID string,
	code fhircodes.IssueTypeCode_Value,
	detailsCode, detailsDisplay string, jobID int) {
	close := metrics.NewChild(ctx, "appendErrorToFile")
	defer close()

	oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, code, detailsCode, detailsDisplay)

	dataDir := conf.GetEnv("FHIR_STAGING_DIR")
	fileName := fmt.Sprintf("%s/%d/%s-error.ndjson", dataDir, jobID, fileUUID)
	/* #nosec -- opening file defined by variable */
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

	if err != nil {
		log.Worker.Error(err)
	}

	defer utils.CloseFileAndLogError(f)
	if _, err := responseutils.WriteOperationOutcome(f, oo); err != nil {
		log.Worker.Error(err)
	}

	// Separate any subsequent error entries
	if _, err := f.WriteString("\n"); err != nil {
		log.Worker.Error(err)
	}
}

func fhirBundleToResourceNDJSON(ctx context.Context, w *bufio.Writer, b *fhirmodels.Bundle, jsonType, beneficiaryID, acoID, fileUUID string, jobID int) {
	close := metrics.NewChild(ctx, "fhirBundleToResourceNDJSON")
	defer close()

	for _, entry := range b.Entries {
		if entry["resource"] == nil {
			continue
		}

		entryJSON, err := json.Marshal(entry["resource"])
		// This is unlikely to happen because we just unmarshalled this data a few lines above.
		if err != nil {
			log.Worker.Error(err)
			appendErrorToFile(ctx, fileUUID, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.InternalErr, fmt.Sprintf("Error marshaling %s to JSON for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), jobID)
			continue
		}
		_, err = w.WriteString(string(entryJSON) + "\n")
		if err != nil {
			log.Worker.Error(err)
			appendErrorToFile(ctx, fileUUID, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.InternalErr, fmt.Sprintf("Error writing %s to file for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), jobID)
		}
	}
}

func checkJobCompleteAndCleanup(ctx context.Context, r repository.Repository, jobID uint) (jobCompleted bool, err error) {
	j, err := r.GetJobByID(ctx, jobID)
	if err != nil {
		return false, err
	}

	switch j.Status {
	case models.JobStatusCompleted:
		return true, nil
	case models.JobStatusCancelled, models.JobStatusFailed:
		// don't update job, Cancelled and Failed are terminal statuses
		log.Worker.Warnf("Failed to mark job as completed (Job %s): %s", j.Status, err)
		return true, nil
	}

	completedCount, err := r.GetJobKeyCount(ctx, jobID)
	if err != nil {
		return false, err
	}

	if completedCount >= j.JobCount {
		staging := fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_STAGING_DIR"), j.ID)
		payload := fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_PAYLOAD_DIR"), j.ID)

		files, err := ioutil.ReadDir(staging)
		if err != nil {
			return false, err
		}

		for _, f := range files {
			oldPath := fmt.Sprintf("%s/%s", staging, f.Name())
			newPath := fmt.Sprintf("%s/%s", payload, f.Name())
			err := os.Rename(oldPath, newPath)
			if err != nil {
				return false, err
			}
		}

		if err = os.Remove(staging); err != nil {
			return false, err
		}

		err = r.UpdateJobStatus(ctx, j.ID, models.JobStatusCompleted)
		if err != nil {
			return false, err
		}
		// Able to mark job as completed
		return true, nil

	}
	// We still have parts of the job that are not complete
	return false, nil
}

func createDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err = os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

type JobError struct {
	ErrorString string
}

func (je JobError) Error() string {
	return je.ErrorString
}

var (
	ErrNoBasePathSet      = JobError{"empty BBBasePath: Must be set"}
	ErrParentJobNotFound  = JobError{"parent job not found"}
	ErrParentJobCancelled = JobError{"parent job cancelled"}
)
