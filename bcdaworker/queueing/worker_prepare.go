/*
Prepare Worker takes all of the arguments of a bulk export request to BCDA API and asynchronously prepares and
creates (enqueues?) all of the subjobs needed for the requests bulk export main job.
*/

package queueing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/bcdaworker/constants"
	"github.com/CMSgov/bcda-app/log"
	m "github.com/CMSgov/bcda-app/middleware"
	"github.com/ccoveille/go-safecast"
	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type PrepareJobArgs struct {
	Job             models.Job
	CMSID           string
	BFDPath         string
	RequestType     service.RequestType
	ResourceTypes   []string
	RequestParamter middleware.RequestParameters
	Since           time.Time
	TransactionID   string
}

func (args PrepareJobArgs) Kind() string {
	return constants.PrepareJobKind
}

// PrepareJobWorker has two BFD clients because it depends on a configuration variable that is not available until Work() is called.
// There were other discussed methods of injecting the client and overwriting the the basepath but ruled out due to the risk and time constraints.
// Many of the Service's functionality is used solely in this PrepareJob functionality and should eventually be migrated when time allows.
type PrepareJobWorker struct {
	river.WorkerDefaults[PrepareJobArgs]
	svc      service.Service
	v1Client client.APIClient
	v2Client client.APIClient
	r        models.Repository
}

func NewPrepareJobWorker() (*PrepareJobWorker, error) {

	logger := log.Worker
	client.SetLogger(logger)

	cfg, err := service.LoadConfig()
	if err != nil {
		logger.Fatalf("failed to load service config. Err: %v", err)
	}

	repository := postgres.NewRepository(database.Connection)
	svc := service.NewService(repository, cfg, "")

	v1, err := client.NewBlueButtonClient(client.NewConfig(constants.BFDV1Path))
	if err != nil {
		logger.Fatalf("failed to load bfd client. Err: %v", err)
		return &PrepareJobWorker{}, err
	}
	v2, err := client.NewBlueButtonClient(client.NewConfig(constants.BFDV2Path))
	if err != nil {
		logger.Fatalf("failed to load bfd client. Err: %v", err)
		return &PrepareJobWorker{}, err
	}

	return &PrepareJobWorker{svc: svc, v1Client: v1, v2Client: v2, r: repository}, nil

}

func (w *PrepareJobWorker) Work(ctx context.Context, rjob *river.Job[PrepareJobArgs]) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
	ctx = context.WithValue(ctx, m.CtxTransactionKey, rjob.Args.TransactionID)
	logger := log.GetCtxLogger(ctx)

	exports, since, err := w.prepareExportJobs(ctx, rjob.Args)
	if err != nil {
		logger.Errorf("failed to add jobs to the main queue: %s", err)
		return err
	}

	client := river.ClientFromContext[pgxv5.Tx](ctx)
	q := riverEnqueuer{client}
	err = w.queueExportJobs(ctx, q, rjob.Args, exports, since)
	if err != nil {
		// TODO update job in jobs table as failed
		logger.Errorf("failed to add jobs to the main queue: %s", err)
		return err
	}

	return nil

}

// prepareExportJobs builds a list of jobs to be processed based on the parent job.
func (p *PrepareJobWorker) prepareExportJobs(ctx context.Context, args PrepareJobArgs) ([]*models.JobEnqueueArgs, time.Time, error) {

	var err error
	exports := []*models.JobEnqueueArgs{}
	logger := log.GetCtxLogger(ctx)

	defer func() {
		if err != nil {
			args.Job.Status = models.JobStatusFailed
		}
		dberr := p.r.UpdateJob(ctx, args.Job)
		if dberr != nil {
			err = fmt.Errorf("%w: %w", err, dberr)
		}
	}()

	id, err := safecast.ToInt(args.Job.ID) // NOSONAR
	if err != nil {                        // NOSONAR
		logger.Error(err)               // NOSONAR
		return exports, args.Since, err // NOSONAR
	} // NOSONAR

	jobData := models.JobEnqueueArgs{
		ID:              id,
		ACOID:           args.Job.ACOID.String(),
		Since:           args.Since.String(),
		TransactionTime: time.Now(),
		CMSID:           args.CMSID,
	}

	args.Job.TransactionTime, err = p.GetBundleLastUpdated(args.BFDPath, jobData)
	if err != nil {
		return exports, args.Since, err
	}

	conditions := service.RequestConditions{
		ReqType:    args.RequestType,
		Resources:  args.ResourceTypes,
		BBBasePath: args.BFDPath,

		CMSID: args.CMSID,
		ACOID: args.Job.ACOID,

		JobID:           args.Job.ID,
		Since:           args.Since,
		TransactionTime: args.Job.TransactionTime,
		CreationTime:    time.Now(),
	}

	exports, err = p.svc.GetQueJobs(ctx, conditions)
	if err != nil {
		logger.Error(err)
		if ok := errors.As(err, &service.CCLFNotFoundError{}); ok {
			return exports, args.Since, err
		} else {
			return exports, args.Since, err
		}
	}
	args.Job.JobCount = len(exports)

	return exports, args.Since, err
}

// GetBundleLastUpdated requests a fake patient in order to acquire the bundle's lastUpdated metadata.
func (p *PrepareJobWorker) GetBundleLastUpdated(basepath string, jobData models.JobEnqueueArgs) (time.Time, error) {
	switch basepath {
	case constants.BFDV1Path:
		b, err := p.v1Client.GetPatient(jobData, "0")
		return b.Meta.LastUpdated, err
	case constants.BFDV2Path:
		b, err := p.v2Client.GetPatient(jobData, "0")
		return b.Meta.LastUpdated, err
	default:
		return time.Time{}, errors.New("no BFD base path")
	}
}

func (p *PrepareJobWorker) queueExportJobs(ctx context.Context, q Enqueuer, args PrepareJobArgs, exports []*models.JobEnqueueArgs, since time.Time) error {
	for _, j := range exports {
		sinceParam := !since.IsZero() || args.RequestType == service.RetrieveNewBeneHistData
		jobPriority := p.svc.GetJobPriority(args.CMSID, j.ResourceType, sinceParam)

		if err := q.AddJob(ctx, *j, int(jobPriority)); err != nil {
			return err
		}
	}
	return nil
}
