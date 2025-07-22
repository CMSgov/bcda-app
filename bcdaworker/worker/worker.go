package worker

import (
	"bufio"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/CMSgov/bcda-app/bcda/cclf/metrics"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/models"
	fhirmodels "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/ccoveille/go-safecast"
	"github.com/sirupsen/logrus"

	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

type Worker interface {
	ValidateJob(ctx context.Context, queJobID int64, jobArgs worker_types.JobEnqueueArgs) (*models.Job, error)
	ProcessJob(ctx context.Context, queJobID int64, job models.Job, jobArgs worker_types.JobEnqueueArgs) error
}

type worker struct {
	r repository.Repository
}

func NewWorker(db *sql.DB) Worker {
	return &worker{postgres.NewRepository(db)}
}

func (w *worker) ValidateJob(ctx context.Context, qjobID int64, jobArgs worker_types.JobEnqueueArgs) (*models.Job, error) {
	if len(jobArgs.BBBasePath) == 0 {
		return nil, ErrNoBasePathSet
	}

	jobID, err := safecast.ToUint(jobArgs.ID)
	if err != nil {
		return nil, err
	}

	exportJob, err := w.r.GetJobByID(ctx, jobID)
	if goerrors.Is(err, repository.ErrJobNotFound) {
		return nil, ErrParentJobNotFound
	} else if err != nil {
		return nil, errors.Wrap(err, "could not retrieve job from database")
	}

	if exportJob.Status == models.JobStatusCancelled {
		return nil, ErrParentJobCancelled
	}

	if exportJob.Status == models.JobStatusFailed {
		return nil, ErrParentJobFailed
	}

	_, err = w.r.GetJobKey(ctx, jobID, qjobID)
	if goerrors.Is(err, repository.ErrJobKeyNotFound) {
		// No job key exists, which means this queue job needs to be processed.
		return exportJob, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "could not retrieve job key from database")
	} else {
		// If there was no error, we found a job key and can avoid re-processing the job.
		return nil, ErrQueJobProcessed
	}
}

func (w *worker) ProcessJob(ctx context.Context, queJobID int64, job models.Job, jobArgs worker_types.JobEnqueueArgs) error {

	t := metrics.GetTimer()
	defer t.Close()
	ctx = metrics.NewContext(ctx, t)
	ctx, c := metrics.NewParent(ctx, fmt.Sprintf("ProcessJob-%s", jobArgs.ResourceType))
	defer c()

	aco, err := w.r.GetACOByUUID(ctx, job.ACOID)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("ProcessJob: could not retrieve ACO from database by UUID %s", job.ACOID))
		return err
	}

	ctx, logger := log.SetCtxLogger(ctx, "cms_id", aco.CMSID)

	err = w.r.UpdateJobStatusCheckStatus(ctx, job.ID, models.JobStatusPending, models.JobStatusInProgress)
	if goerrors.Is(err, repository.ErrJobNotUpdated) {
		// could also occur if job was marked as cancelled
		logger.Warnf("Failed to update job. Assume job already updated. Continuing. %s", err.Error())
	} else if err != nil {
		err = errors.Wrap(err, "ProcessJob: could not update job status in database")
		logger.Error(err)
		return err
	}

	bb, err := client.NewBlueButtonClient(client.NewConfig(jobArgs.BBBasePath))
	if err != nil {
		err = errors.Wrap(err, "ProcessJob: could not create Blue Button client")
		logger.Error(err)
		return err
	}

	jobID := strconv.Itoa(jobArgs.ID)
	//temp Job path is not dependent upon the directory. Using a UUID for a directory string prevents race conditions.
	tempJobPath := fmt.Sprintf("%s/%s", conf.GetEnv("FHIR_TEMP_DIR"), uuid.NewRandom())
	stagingPath := fmt.Sprintf("%s/%s", conf.GetEnv("FHIR_STAGING_DIR"), jobID)
	payloadPath := fmt.Sprintf("%s/%s", conf.GetEnv("FHIR_PAYLOAD_DIR"), jobID)

	// Create a temporary path for the job files before they go into the staging directory
	if err = createDir(tempJobPath); err != nil {
		err = errors.Wrap(err, fmt.Sprintf("ProcessJob: could not create temporary directory on worker for jobID %s", jobID))
		logger.Error(err)
		return err
	}
	defer os.RemoveAll(tempJobPath)

	if err = createDir(stagingPath); err != nil {
		err = errors.Wrap(err, fmt.Sprintf("ProcessJob: could not create FHIR staging path directory for jobID %s", jobID))
		logger.Error(err)
		return err
	}

	// Create directory for job results.
	// This will be used in the clean up later to move over processed files.
	if err = createDir(payloadPath); err != nil {
		err = errors.Wrap(err, "ProcessJob: could not create FHIR payload path directory")
		logger.Error(err)
		return err
	}

	jobKeys, err := writeBBDataToFile(ctx, w.r, bb, *aco.CMSID, queJobID, jobArgs, tempJobPath)

	// This is only run AFTER completion of all the collection
	if err != nil {
		logger.Error(errors.Wrap(err, "ProcessJob: Error occurred when writing BFD Data to file"))

		// only inProgress jobs should move to a failed status (i.e. don't move a cancelled job to failed)
		err = w.r.UpdateJobStatusCheckStatus(ctx, job.ID, models.JobStatusInProgress, models.JobStatusFailed)
		if goerrors.Is(err, repository.ErrJobNotUpdated) {
			jobUpdateFailMessage := fmt.Sprintf("ProcessJob: Failed to update job (job cancelled): %s", err)
			logger.Warn(jobUpdateFailMessage)
			return errors.Wrap(err, jobUpdateFailMessage)
		} else if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("Error updating the job status to %s", models.JobStatusFailed))
			logger.Error(err)
			return err
		} else {
			logger.Error("Job failed. Job ID: ", job.ID)
		}
	}
	//move the files over
	err = compressFiles(ctx, tempJobPath, stagingPath)
	if err != nil {
		logger.Error(err)
		return err
	}

	err = createJobKeys(ctx, w.r, jobKeys, job.ID)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func compressFiles(ctx context.Context, tempDir string, stagingDir string) error {
	logger := log.GetCtxLogger(ctx)
	// Open the input file
	files, err := os.ReadDir(tempDir)
	if err != nil {
		err = errors.Wrap(err, "Error reading from the staging directory for files for Job")
		return err
	}
	gzipLevel, err := strconv.Atoi(os.Getenv("COMPRESSION_LEVEL"))
	if err != nil || gzipLevel < 1 || gzipLevel > 9 { //levels 1-9 supported by BCDA.
		gzipLevel = gzip.DefaultCompression
		logger.Warnf("COMPRESSION_LEVEL not set to appropriate value; using default.")
	}
	for _, f := range files {
		oldPath := fmt.Sprintf("%s/%s", tempDir, f.Name())
		newPath := fmt.Sprintf("%s/%s", stagingDir, f.Name())
		//Anonymous function to ensure defer statements run
		err := func() error {
			inputFile, err := os.Open(filepath.Clean(oldPath))
			if err != nil {
				return err
			}
			defer CloseOrLogError(logger, inputFile)

			outputFile, err := os.Create(filepath.Clean(newPath))
			if err != nil {
				return err
			}
			defer CloseOrLogError(logger, outputFile)
			gzipWriter, err := gzip.NewWriterLevel(outputFile, gzipLevel)
			if err != nil {
				return err
			}
			defer gzipWriter.Close()

			// Copy the data from the input file to the gzip writer
			if _, err := io.Copy(gzipWriter, inputFile); err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}

	}
	return nil

}

func CloseOrLogError(logger logrus.FieldLogger, f *os.File) {
	if f == nil {
		return
	}
	if err := f.Close(); err != nil {
		logger.Warnf("Error closing file: %v", err)
	}
}

// writeBBDataToFile sends requests to BlueButton and writes the results to ndjson files.
// A list of JobKeys are returned, containing the names of files that were created.
// Filesnames can be "blank.ndjson", "<uuid>.ndjson", or "<uuid>-error.ndjson".
func writeBBDataToFile(ctx context.Context, r repository.Repository, bb client.APIClient,
	cmsID string, queJobID int64, jobArgs worker_types.JobEnqueueArgs, tmpDir string) (jobKeys []models.JobKey, err error) {

	id, err := safecast.ToUint(jobArgs.ID)
	if err != nil {
		return nil, err
	}
	jobKeys = append(jobKeys, models.JobKey{JobID: id, QueJobID: &queJobID, FileName: models.BlankFileName, ResourceType: jobArgs.ResourceType})

	logger := log.GetCtxLogger(ctx)
	close := metrics.NewChild(ctx, "writeBBDataToFile")
	defer close()

	var bundleFunc func(bene models.CCLFBeneficiary) (*fhirmodels.Bundle, error)
	// NOTE: Currently all Coverage/EOB/Patient requests are for adjudicated data and
	// Claim/ClaimResponse are partially-adjudicated, future work may require checking what
	// kind of backing data to pull from if there is overlap (one or more FHIR resource
	// used for representing both adjudicated and partially-adjudicated data)
	switch jobArgs.ResourceType {
	case "Coverage":
		bundleFunc = func(bene models.CCLFBeneficiary) (*fhirmodels.Bundle, error) {
			return bb.GetCoverage(jobArgs, bene.BlueButtonID)
		}
	case "ExplanationOfBenefit":
		bundleFunc = func(bene models.CCLFBeneficiary) (*fhirmodels.Bundle, error) {
			cw := client.ClaimsWindow{
				LowerBound: jobArgs.ClaimsWindow.LowerBound,
				UpperBound: jobArgs.ClaimsWindow.UpperBound}
			return bb.GetExplanationOfBenefit(jobArgs, bene.BlueButtonID, cw)
		}
	case "Patient":
		bundleFunc = func(bene models.CCLFBeneficiary) (*fhirmodels.Bundle, error) {
			return bb.GetPatient(jobArgs, bene.BlueButtonID)
		}
		//NOTE: The assumption is Claim/ClaimResponse is always partially-adjudicated, future work may require checking what
		//kind of backing data to pull from
	case "Claim":
		bundleFunc = func(bene models.CCLFBeneficiary) (*fhirmodels.Bundle, error) {
			cw := client.ClaimsWindow{
				LowerBound: jobArgs.ClaimsWindow.LowerBound,
				UpperBound: jobArgs.ClaimsWindow.UpperBound}
			return bb.GetClaim(jobArgs, bene.MBI, cw)
		}
	case "ClaimResponse":
		bundleFunc = func(bene models.CCLFBeneficiary) (*fhirmodels.Bundle, error) {
			cw := client.ClaimsWindow{
				LowerBound: jobArgs.ClaimsWindow.LowerBound,
				UpperBound: jobArgs.ClaimsWindow.UpperBound}
			return bb.GetClaimResponse(jobArgs, bene.MBI, cw)
		}
	default:
		return jobKeys, fmt.Errorf("unsupported resource type requested: %s", jobArgs.ResourceType)
	}

	fileUUID := uuid.New()
	f, err := os.Create(fmt.Sprintf("%s/%s.ndjson", tmpDir, fileUUID))
	if err != nil {
		err = errors.Wrap(err, "Error creating ndjson file")
		return jobKeys, err
	}

	defer utils.CloseFileAndLogError(f)

	w := bufio.NewWriter(f)
	defer w.Flush()
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

		fileErrMsg, code, err := func() (string, fhircodes.IssueTypeCode_Value, error) {
			id, err := strconv.ParseUint(beneID, 10, 64)
			if err != nil {
				return fmt.Sprintf("Error failed to convert %s to uint", beneID), fhircodes.IssueTypeCode_EXCEPTION, err
			}

			// NOTE: with adjudicated data sets, we first need to lookup the Patient ID
			// before gathering EOB/Coverage results; however with partially-adjudicated data
			// that is not yet possible because their are no Patient FHIR resources. This
			// boolean indicates whether or not we need to skip that lookup step
			fetchBBId := !utils.ContainsString([]string{"Claim", "ClaimResponse"}, jobArgs.ResourceType)
			bene, err := getBeneficiary(ctx, r, uint(id), bb, fetchBBId, jobArgs)
			if err != nil {
				//MBI is appended inside file, not printed out to system logs
				return fmt.Sprintf("Error retrieving BlueButton ID for cclfBeneficiary MBI %s", bene.MBI), fhircodes.IssueTypeCode_NOT_FOUND, err
			}

			b, err := bundleFunc(bene)
			if err != nil {
				//MBI is appended inside file, not printed out to system logs
				return fmt.Sprintf("Error retrieving %s for beneficiary MBI %s in ACO %s", jobArgs.ResourceType, bene.MBI, jobArgs.ACOID), fhircodes.IssueTypeCode_NOT_FOUND, err
			}
			fhirBundleToResourceNDJSON(ctx, w, b, jobArgs.ResourceType, beneID, cmsID, fileUUID, tmpDir)
			return "", 0, nil
		}()

		if err != nil {
			logger.Error(err)
			errorCount++
			appendErrorToFile(ctx, fileUUID, code, responseutils.BbErr, fileErrMsg, tmpDir)
		}

		failPct := (float64(errorCount) / totalBeneIDs) * 100
		if failPct >= failThreshold {
			failed = true
			break
		}
	}

	if err = w.Flush(); err != nil {
		return jobKeys, errors.Wrap(err, "Error in writing the buffered data to the writer")
	}

	if failed {
		if ctx.Err() == context.Canceled {
			appendErrorToFile(ctx, fileUUID, fhircodes.IssueTypeCode_PROCESSING, responseutils.BbErr, "Parent job was cancelled", tmpDir)
			return jobKeys, errors.New("Parent job was cancelled")
		}
		return jobKeys, errors.New(fmt.Sprintf("Number of failed requests has exceeded threshold of %f ", failThreshold))
	}

	fstat, err := f.Stat()
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Error in obtaining FileInfo structure describing the file for ndjson fileUUID %s jobId %d for cmsID %s", fileUUID, jobArgs.ID, cmsID))
		return jobKeys, err
	}

	if fstat.Size() != 0 {
		pr := &jobKeys[0]
		(*pr).FileName = fileUUID + ".ndjson"
	}

	if errorCount > 0 {
		jobKeys = append(jobKeys, models.JobKey{JobID: id, QueJobID: &queJobID, FileName: fileUUID + "-error.ndjson", ResourceType: jobArgs.ResourceType})
	}
	return jobKeys, nil
}

// getBeneficiary returns the beneficiary. The bb ID value is retrieved and set in the model.
func getBeneficiary(ctx context.Context, r repository.Repository, beneID uint, bb client.APIClient, fetchBBId bool, jobData worker_types.JobEnqueueArgs) (models.CCLFBeneficiary, error) {
	bene, err := r.GetCCLFBeneficiaryByID(ctx, beneID)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Error retrieving cclfBeneficiary record by cclfBeneficiaryId %d", beneID))
		return models.CCLFBeneficiary{}, err
	}

	cclfBeneficiary := *bene

	if fetchBBId {
		bbID, err := getBlueButtonID(bb, cclfBeneficiary.MBI, jobData)

		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("failed to get blueButtonId for cclfBeneficiaryId %d", beneID))
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
	detailsCode, detailsDisplay string, tempDir string) {
	close := metrics.NewChild(ctx, "appendErrorToFile")
	defer close()

	logger := log.GetCtxLogger(ctx)
	rw := responseutils.NewResponseWriter()
	oo := rw.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, code, detailsCode, detailsDisplay)

	fileName := fmt.Sprintf("%s/%s-error.ndjson", tempDir, fileUUID)
	/* #nosec -- opening file defined by variable */
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

	if err != nil {
		err = errors.Wrap(err, "Unable to append error to file: OS error encountered while opening the file")
		logger.Error(err)
		return
	}

	defer utils.CloseFileAndLogError(f)
	if _, err := rw.WriteOperationOutcome(f, oo); err != nil {
		err = errors.Wrap(err, "Issue during append error to file: Error encountered during WriteOperationalOutcome")
		logger.Error(err)
	}

	// Separate any subsequent error entries
	if _, err := f.WriteString("\n"); err != nil {
		err = errors.Wrap(err, "Issue during append error to file: Unable to write new line separator ")
		logger.Error(err)
	}
}

func fhirBundleToResourceNDJSON(ctx context.Context, w *bufio.Writer, b *fhirmodels.Bundle, jsonType, beneficiaryID, acoID, fileUUID string, tmpDir string) {
	close := metrics.NewChild(ctx, "fhirBundleToResourceNDJSON")
	defer close()
	defer w.Flush()
	logger := log.GetCtxLogger(ctx)
	for _, entry := range b.Entries {
		if entry["resource"] == nil {
			continue
		}

		entryJSON, err := json.Marshal(entry["resource"])
		// This is unlikely to happen because we just unmarshalled this data a few lines above.
		if err != nil {
			logger.Error(err)
			appendErrorToFile(ctx, fileUUID, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.InternalErr, fmt.Sprintf("Error marshaling %s to JSON for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), tmpDir)
			continue
		}

		_, err = w.Write(append(entryJSON, '\n'))
		if err != nil {
			logger.Error(err)
			appendErrorToFile(ctx, fileUUID, fhircodes.IssueTypeCode_EXCEPTION,
				responseutils.InternalErr, fmt.Sprintf("Error writing %s to file for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), tmpDir)
		}
	}
}

func CheckJobCompleteAndCleanup(ctx context.Context, r repository.Repository, jobID uint) (jobCompleted bool, err error) {
	logger := log.GetCtxLogger(ctx)
	j, err := r.GetJobByID(ctx, jobID)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Failed retrieve job by id (Job %d)", jobID))
		return false, err
	}

	switch j.Status {
	case models.JobStatusCompleted:
		return true, nil
	case models.JobStatusCancelled, models.JobStatusFailed:
		// don't update job, Cancelled and Failed are terminal statuses
		logger.Warnf("Failed to mark job as completed (Job %s)", j.Status)
		return true, nil
	}

	completedCount, err := r.GetJobKeyCount(ctx, jobID)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Failed get job key count (Job %d)", jobID))
		return false, err
	}

	logger.Debugf("completedCount: %d jobCount: %d", completedCount, j.JobCount)

	if completedCount >= j.JobCount {
		staging := fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_STAGING_DIR"), j.ID)
		payload := fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_PAYLOAD_DIR"), j.ID)

		files, err := os.ReadDir(staging)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("Error reading from the staging directory for files for Job %d", jobID))
			return false, err
		}

		for _, f := range files {
			oldPath := fmt.Sprintf("%s/%s", staging, f.Name())
			newPath := fmt.Sprintf("%s/%s", payload, f.Name())
			err := os.Rename(oldPath, newPath)
			if err != nil {
				err = errors.Wrap(err, fmt.Sprintf("Error moving the file %s from staging to payload directory", f.Name()))
				return false, err
			}
		}

		if err = os.Remove(staging); err != nil {
			err = errors.Wrap(err, "Error removing the staging directory")
			return false, err
		}

		err = r.UpdateJobStatus(ctx, j.ID, models.JobStatusCompleted)
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("Error updating the job status to %s for job id %d", models.JobStatusCompleted, j.ID))
			return false, err
		}
		// Able to mark job as completed
		return true, nil

	}
	// We still have parts of the job that are not complete
	return false, nil
}

func createJobKeys(ctx context.Context, r repository.Repository, jobKeys []models.JobKey, id uint) error {
	if err := r.CreateJobKeys(ctx, jobKeys); err != nil {
		filenames := ""
		for _, jobKey := range jobKeys {
			filenames += " " + jobKey.FileName
		}
		err = errors.Wrap(err, fmt.Sprintf("Error creating job key records for filenames%s", filenames))
		return err
	}

	_, err := CheckJobCompleteAndCleanup(ctx, r, id)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Error checking job completion & cleanup for job id %d", id))
		return err
	}

	return nil
}

func createDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err = os.MkdirAll(path, 0744); err != nil {
			return err
		}
		return err
	} else if err != nil {
		return err
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
	ErrParentJobFailed    = JobError{"parent job failed"}
	ErrQueJobProcessed    = JobError{"que job already processed"}
)
