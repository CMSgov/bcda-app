package worker

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/google/fhir/go/jsonformat"
	"github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
	"github.com/huandu/go-sqlbuilder"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

type AlrWorker interface {
	ProcessAlrJob(ctx context.Context, jobArgs models.JobAlrEnqueueArgs) error
	UpdateJobAlrStatus(ctx context.Context, jobID uint, newStatus models.JobStatus) error
	GetAlrJobStatus(ctx context.Context, jobID uint) (*models.JobStatus, error)
}

type alrWorker struct {
	*postgres.AlrRepository
	FHIR_STAGING_DIR string
}

const (
	sqlFlavor = sqlbuilder.PostgreSQL
)

func NewAlrWorker(db *sql.DB) *alrWorker {
	alrR := postgres.NewAlrRepo(db)

	// embed data struct that has method GetAlr
	worker := &alrWorker{
		AlrRepository:    alrR,
		FHIR_STAGING_DIR: "",
	}

	err := conf.Checkout(worker) // worker is already a reference, no & needed
	if err != nil {
		logrus.Fatal("Could not get data from conf for ALR.", err)
	}

	return worker
}

func (a *alrWorker) ProcessAlrJob(
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
	fileUUID := uuid.New()
	f, err := os.Create(fmt.Sprintf("%s/%d/%s.ndjson", a.FHIR_STAGING_DIR,
		id, fileUUID))
	if err != nil {
		logrus.Error(err)
		return err
	}
	w := bufio.NewWriter(f)
	defer utils.CloseFileAndLogError(f)

	// Serialize data into JSON
	marshaller, err := jsonformat.NewPrettyMarshaller(jsonformat.STU3)
	if err != nil {
		logrus.Error(err)
		return err
	}

	// data is what will be sent over the channel to the go routine
	type data struct {
		patient      *resources_go_proto.Patient
		observations []*resources_go_proto.Observation
	}

	// Creating channels for go routine
	// c is buffered b/c IO operation is slower than unmarshalling
	c := make(chan data, 1000) // 1000 rows before blocking
	result := make(chan error)

	// Go rountine + closure that will write to file
	// Two channels involved with this closure. One receiving "data", and the
	// other will send an error is there was any error. The worker thread will
	// will also use the result channel to wait for the IO to finish.
	go func(c chan data, w *bufio.Writer, result chan error) {
		// Receive data through channel

		for i := range c {
			// marshall
			patientb, err := marshaller.Marshal(i.patient)
			patients := string(patientb) + "\n"
			if err != nil {
				// Make sure to send err back to the other thread
				result <- err
				return
			}
			var observations []string

			for _, observation := range i.observations {
				obsMarshalled, err := marshaller.Marshal(observation)
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

		// Everything went a-ok... send nil
		result <- nil

	}(c, w, result)

	// Marshall into JSON and send it over the channel
	for i := range alrModels {
		patient, observations := alr.ToFHIR(&alrModels[i], alrModels[i].Timestamp)
		c <- data{patient, observations}
	}

	// Wait on the go routine and return back results
	return <-result
}

func (a *alrWorker) UpdateJobAlrStatus(ctx context.Context, jobID uint,
	newStatus models.JobStatus) error {

	ub := sqlFlavor.NewUpdateBuilder().Update("jobs")
	ub.Set(ub.Assign("updated_at", sqlbuilder.Raw("NOW()")))
	ub.SetMore(ub.Assign("status", newStatus))
	ub.Where(ub.Equal("id", jobID))

	query, args := ub.Build()
	result, err := a.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return repository.ErrJobNotUpdated
	}

	return nil
}

func (a *alrWorker) GetAlrJobStatus(ctx context.Context, jobID uint) (*models.JobStatus,
	error) {
	sb := sqlFlavor.NewSelectBuilder()
	sb.Select("status")
	sb.From("jobs").Where(sb.Equal("id", jobID))

	query, args := sb.Build()

	var j models.JobStatus

	err := a.QueryRowContext(ctx, query, args...).Scan(&j)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrJobNotFound
		}
		return nil, err
	}

	return &j, nil
}
