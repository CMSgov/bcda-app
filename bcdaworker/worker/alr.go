package worker

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	workerpg "github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/google/fhir/go/jsonformat"
	"github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
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

//data is used to send FHIR data to a go routine to write to disk
type data struct {
	patient      *resources_go_proto.Patient
	observations []*resources_go_proto.Observation
}

var resources = [...]string{"patient", "observations"}

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
		logrus.Fatal("Could not get data from conf for ALR.", err)
	}

	return worker
}

func goWriter(ctx context.Context, a *AlrWorker, c chan data, fileMap map[string]*os.File,
	marshaller *jsonformat.Marshaller, result chan error, resourceTypes []string, id uint) {

	writerPool := make([]*bufio.Writer, len(fileMap))

	for i, n := range resourceTypes {
		file := fileMap[n]
		w := bufio.NewWriter(file)
		writerPool[i] = w
		defer utils.CloseFileAndLogError(file)
	}

	for i := range c {
		// marshall
		patientb, err := marshaller.MarshalResource(i.patient)
		patients := string(patientb) + "\n"
		if err != nil {
			// Make sure to send err back to the other thread
			result <- err
			return
		}
		var observations []string

		for _, observation := range i.observations {
			obsMarshalled, err := marshaller.MarshalResource(observation)
			if err != nil {
				result <- err
				return
			}
			observations = append(observations, string(obsMarshalled))
		}

		observation := strings.Join(observations, "\n")

		alrResources := []string{patients, observation}

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

	// update the jobs keys
	for resource, path := range fileMap {
		filename := filepath.Base(path.Name())
		jk := models.JobKey{JobID: id, FileName: filename, ResourceType: resource}
		if err := a.Repository.CreateJobKey(ctx, jk); err != nil {
			result <- fmt.Errorf("failed to create job key: %w", err)
			return
		}
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
	jobArgs models.JobAlrEnqueueArgs,
) error {

	// Parse the jobAlrEnqueueArgs
	aco := jobArgs.CMSID
	id := jobArgs.ID
	MBIs := jobArgs.MBIs
	lowerBound := jobArgs.LowerBound
	upperBound := jobArgs.UpperBound

	// Pull the data from ALR tables (alr & alr_meta)
	alrModels, err := a.GetAlr(ctx, aco, MBIs, lowerBound, upperBound)
	if err != nil {
		logrus.Error(err)
		return err
	}

	// Set up IO operation to dump ndjson

	// Created necessary directory
	err = os.MkdirAll(fmt.Sprintf("%s/%d", a.StagingDir, id), 0744)
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
			logrus.Error(err)
			return err
		}

	}

	// Serialize data into JSON
	marshaller, err := jsonformat.NewMarshaller(false, "", "", jsonformat.STU3)
	if err != nil {
		logrus.Error(err)
		return err
	}

	// Creating channels for go routine
	// c is buffered b/c IO operation is slower than unmarshalling
	c := make(chan data, 1000) // 1000 rows before blocking
	result := make(chan error)

	// A go routine that will streamed data to write to disk.
	// Reason for a go routine is to not block when writing, since disk writing is
	// generally slower than memory access. We are streaming to keep mem lower.
	go goWriter(ctx, a, c, fileMap, marshaller, result, resources[:], id)

	// Marshall into JSON and send it over the channel
	for i := range alrModels {
		patient, observations := alr.ToFHIR(&alrModels[i], alrModels[i].Timestamp)
		c <- data{patient, observations}
	}

	// close channel c since we are no longer writing to it
	close(c)

	// Wait on the go routine to finish
	if err := <-result; err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	// If we did not have an ALR data to write, we'll write a specific file name that indicates that the
	// there is no data associated with this job.
	if len(alrModels) == 0 {
		jk := models.JobKey{JobID: id, FileName: models.BlankFileName, ResourceType: "ALR"}
		if err := a.Repository.CreateJobKey(ctx, jk); err != nil {
			return fmt.Errorf("failed to create job key: %w", err)
		}
	}

	return nil

}
