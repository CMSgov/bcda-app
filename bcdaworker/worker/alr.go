package worker

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/utils"
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
		- Use to send FHIR data to a go routine to write to disk
******************************************************************************/

type AlrWorker struct {
	*postgres.AlrRepository
	FHIR_STAGING_DIR string
	FHIR_PAYLOAD_DIR string
	NdjsonFilename   string
}

type data struct {
	patient      *resources_go_proto.Patient
	observations []*resources_go_proto.Observation
}

/******************************************************************************
	Functions
	-NewAlrWorker
	-goWriter
******************************************************************************/

func NewAlrWorker(db *sql.DB) AlrWorker {
	alrR := postgres.NewAlrRepo(db)

	// embed data struct that has method GetAlr
	worker := AlrWorker{
		AlrRepository:    alrR,
		FHIR_STAGING_DIR: "",
		NdjsonFilename:   "", // Filled in later
	}

	err := conf.Checkout(&worker) // worker is already a reference, no & needed
	if err != nil {
		logrus.Fatal("Could not get data from conf for ALR.", err)
	}

	return worker
}

func goWriter(c chan data, w *bufio.Writer,
	marshaller *jsonformat.Marshaller, result chan error) {

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

		// IO operation
		_, err = w.WriteString(patients + observation)
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
	err = createDir(fmt.Sprintf("%s/%d", a.FHIR_STAGING_DIR, id))
	if err != nil {
		return err
	}

	a.NdjsonFilename = uuid.New()
	f, err := os.Create(fmt.Sprintf("%s/%d/%s.ndjson", a.FHIR_STAGING_DIR,
		id, a.NdjsonFilename))
	if err != nil {
		logrus.Error(err)
		return err
	}
	w := bufio.NewWriter(f)
	defer utils.CloseFileAndLogError(f)

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
	go goWriter(c, w, marshaller, result)

	// Marshall into JSON and send it over the channel
	for i := range alrModels {
		patient, observations := alr.ToFHIR(&alrModels[i], alrModels[i].Timestamp)
		c <- data{patient, observations}
	}

	// close channel c since we no longer writing to it
	close(c)

	// Wait on the go routine to finish
	return <-result
}
