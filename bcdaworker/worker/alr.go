package worker

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	workerpg "github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	workerutils "github.com/CMSgov/bcda-app/bcdaworker/worker/utils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/pborman/uuid"
)

/******************************************************************************
	Data Structures
	-AlrWorker
	-data
******************************************************************************/

type AlrWorker struct {
	*postgres.AlrRepository
	repository.Repository
	StagingDir string `conf:"FHIR_STAGING_DIR"`
}

var resources = [...]string{"patient", "coverage", "group", "risk", "observations", "covidEpisode"}

/******************************************************************************
	Functions
	-NewAlrWorker
	-goWriter
******************************************************************************/

func NewAlrWorker(db *sql.DB) AlrWorker {
	alrR := postgres.NewAlrRepo(db)

	// embed data struct that has method GetAlr
	worker := AlrWorker{
		AlrRepository: alrR,
		Repository:    workerpg.NewRepository(db),
	}

	err := conf.Checkout(&worker)
	if err != nil {
		log.Worker.Fatal("Could not get data from conf for ALR.", err)
	}

	return worker
}

func goWriterV1(ctx context.Context, a *AlrWorker, c chan *alr.AlrFhirBulk, fileMap map[string]*os.File,
	result chan error, resourceTypes []string, id uint, queJobID int64) {

	writerPool := make([]*bufio.Writer, len(fileMap))

	for i, n := range resourceTypes {
		file := fileMap[n]
		w := bufio.NewWriter(file)
		writerPool[i] = w
		defer utils.CloseFileAndLogError(file)
	}

	for i := range c {
		// marshalling structs into JSON
		for j := range i.V1 {

			alrResources, err := i.V1[j].FhirToString()
			if err != nil {
				result <- err
				return
			}

			if len(alrResources) != len(writerPool) {
				panic(fmt.Sprintf("Writer %d, fileMap %d, alrR %d", len(writerPool), len(fileMap), len(alrResources)))
			}

			// IO operations
			for n, resource := range alrResources {

				w := writerPool[n]

				_, err = w.WriteString(resource)
				if err != nil {
					result <- err
					return
				}
				err = w.Flush()
				if err != nil {
					result <- err
					return
				}

			}
		}
	}

	// update the jobs keys
	var jobKeys []models.JobKey
	for resource, path := range fileMap {
		filename := filepath.Base(path.Name())
		jk := models.JobKey{JobID: id, QueJobID: &queJobID, FileName: filename, ResourceType: resource}
		jobKeys = append(jobKeys, jk)
	}

	if err := a.Repository.CreateJobKeys(ctx, jobKeys); err != nil {
		result <- fmt.Errorf(constants.JobKeyCreateErr, err)
		return
	}

	result <- nil
}

func goWriterV2(ctx context.Context, a *AlrWorker, c chan *alr.AlrFhirBulk, fileMap map[string]*os.File,
	result chan error, resourceTypes []string, id uint, queJobID int64) {

	writerPool := make([]*bufio.Writer, len(fileMap))

	for i, n := range resourceTypes {
		file := fileMap[n]
		w := bufio.NewWriter(file)
		writerPool[i] = w
		defer utils.CloseFileAndLogError(file)
	}

	for i := range c {
		// marshalling structs into JSON
		for j := range i.V2 {

			alrResources, err := i.V2[j].FhirToString()
			if err != nil {
				result <- err
				return
			}

			if len(alrResources) != len(writerPool) {
				panic(fmt.Sprintf("Writer %d, fileMap %d, alrR %d", len(writerPool), len(fileMap), len(alrResources)))
			}

			// IO operations
			for n, resource := range alrResources {

				w := writerPool[n]

				_, err = w.WriteString(resource)
				if err != nil {
					result <- err
					return
				}
				err = w.Flush()
				if err != nil {
					result <- err
					return
				}

			}
		}
	}

	// update the jobs keys
	var jobKeys []models.JobKey
	for resource, path := range fileMap {
		filename := filepath.Base(path.Name())
		jk := models.JobKey{JobID: id, QueJobID: &queJobID, FileName: filename, ResourceType: resource}
		jobKeys = append(jobKeys, jk)
	}

	if err := a.Repository.CreateJobKeys(ctx, jobKeys); err != nil {
		result <- fmt.Errorf(constants.JobKeyCreateErr, err)
		return
	}

	result <- nil
}

/******************************************************************************
	Methods
	-ProcessAlrJob
******************************************************************************/

// ProcessAlrJob is a function called by the Worker to serve ALR data to users
func (a *AlrWorker) ProcessAlrJob(
	ctx context.Context,
	queJobID int64,
	jobArgs models.JobAlrEnqueueArgs,
) error {

	// Parse the jobAlrEnqueueArgs
	id := jobArgs.ID
	MBIs := jobArgs.MBIs
	BBBasePath := jobArgs.BBBasePath
	MetaKey := jobArgs.MetaKey

	// Pull the data from ALR tables (alr & alr_meta)
	alrModels, err := a.GetAlr(ctx, MetaKey, MBIs)
	if err != nil {
		log.Worker.Error(err)
		return err
	}

	// If we did not have any ALR data to write, we'll write a specific file name that indicates that
	// there is no data associated with this job.
	if len(alrModels) == 0 {
		jk := models.JobKey{JobID: id, QueJobID: &queJobID, FileName: models.BlankFileName, ResourceType: "ALR"}
		if err := a.Repository.CreateJobKey(ctx, jk); err != nil {
			return fmt.Errorf(constants.JobKeyCreateErr, err)
		}
	}

	// Set up IO operation to dump ndjson
	// Created necessary directory
	err = os.MkdirAll(fmt.Sprintf("%s/%d", a.StagingDir, id), 0750)
	if err != nil {
		return err
	}

	// Get the number FHIR resource types we are using for ALR
	// This is temporary until the resource types become more permanent
	fieldNum := len(resources)
	fileMap := make(map[string]*os.File, fieldNum)

	for i := 0; i < fieldNum; i++ {

		ndjsonFilename := uuid.New()
		f, err := os.Create(fmt.Sprintf("%s/%d/%s.ndjson", a.StagingDir,
			id, ndjsonFilename))

		fileMap[resources[i]] = f

		if err != nil {
			log.Worker.Error(err)
			return err
		}

	}

	// Creating channels for go routine
	// c is buffered b/c IO operation is slower than unmarshalling
	c := make(chan *alr.AlrFhirBulk, 1000) // 1000 rows before blocking
	result := make(chan error)

	// A go routine that will streamed data to write to disk.
	// Reason for a go routine is to not block when writing, since disk writing is
	// generally slower than memory access. We are streaming to keep mem lower.
	if jobArgs.BBBasePath == "/v1/fhir" {
		go goWriterV1(ctx, a, c, fileMap, result, resources[:], id, queJobID)
	} else {
		go goWriterV2(ctx, a, c, fileMap, result, resources[:], id, queJobID)
	}

	// Marshall into JSON and send it over the channel
	const Limit = 100

	workerutils.AlrSlicer(alrModels, c, Limit, BBBasePath)

	// Wait on the go routine to finish
	if err := <-result; err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil

}
