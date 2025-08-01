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
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/log"
	m "github.com/CMSgov/bcda-app/middleware"
	"github.com/ccoveille/go-safecast"
	pgxv5 "github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// PrepareJobWorker has two BFD clients because it depends on a configuration variable that is not available until Work() is called.
// There were other discussed methods of injecting the client and overwriting the the basepath but ruled out due to the risk and time constraints.
// Many of the Service's functionality is used solely in this PrepareJob functionality and should eventually be migrated when time allows.
type PrepareJobWorker struct {
	river.WorkerDefaults[worker_types.PrepareJobArgs]
	svc      service.Service
	v1Client client.APIClient
	v2Client client.APIClient
	v3Client client.APIClient
	r        models.Repository
}

func NewPrepareJobWorker() (*PrepareJobWorker, error) {

	logger := log.Worker
	client.SetLogger(logger)

	cfg, err := service.LoadConfig()
	if err != nil {
		logger.Fatalf("failed to load service config. Err: %v", err)
	}
	if len(cfg.ACOConfigs) == 0 {
		logger.Fatalf("no ACO configs found, these are required for downstream processing")
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
	v3, err := client.NewBlueButtonClient(client.NewConfig(constants.BFDV3Path))
	if err != nil {
		logger.Fatalf("failed to load bfd client. Err: %v", err)
		return &PrepareJobWorker{}, err
	}

	return &PrepareJobWorker{svc: svc, v1Client: v1, v2Client: v2, v3Client: v3, r: repository}, nil

}

func (w *PrepareJobWorker) Work(ctx context.Context, rjob *river.Job[worker_types.PrepareJobArgs]) error {

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
func (p *PrepareJobWorker) prepareExportJobs(ctx context.Context, args worker_types.PrepareJobArgs) ([]*worker_types.JobEnqueueArgs, time.Time, error) {

	var err error
	exports := []*worker_types.JobEnqueueArgs{}
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

	id, err := safecast.ToInt(args.Job.ID)
	if err != nil {
		logger.Error(err)
		return exports, args.Since, err
	}

	jobData := worker_types.JobEnqueueArgs{
		ID:              id,
		ACOID:           args.Job.ACOID.String(),
		Since:           args.Since.String(),
		TypeFilter:      args.TypeFilter,
		TransactionTime: time.Now(),
		CMSID:           args.CMSID,
	}

	args.Job.TransactionTime, err = p.GetBundleLastUpdated(args.BFDPath, jobData)
	if err != nil {
		return exports, args.Since, err
	}

	exports, err = p.svc.GetQueJobs(ctx, args)
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
func (p *PrepareJobWorker) GetBundleLastUpdated(basepath string, jobData worker_types.JobEnqueueArgs) (time.Time, error) {
	switch basepath {
	case constants.BFDV1Path:
		b, err := p.v1Client.GetPatient(jobData, "0")
		return b.Meta.LastUpdated, err
	case constants.BFDV2Path:
		b, err := p.v2Client.GetPatient(jobData, "0")
		return b.Meta.LastUpdated, err
	case constants.BFDV3Path:
		return jobData.TransactionTime, nil // TODO: V3
	default:
		return time.Time{}, errors.New("no BFD base path")
	}
}

func (p *PrepareJobWorker) queueExportJobs(ctx context.Context, q Enqueuer, args worker_types.PrepareJobArgs, exports []*worker_types.JobEnqueueArgs, since time.Time) error {
	for _, j := range exports {
		sinceParam := !since.IsZero() || args.RequestType == constants.RetrieveNewBeneHistData
		jobPriority := p.svc.GetJobPriority(args.CMSID, j.ResourceType, sinceParam)

		if err := q.AddJob(ctx, *j, int(jobPriority)); err != nil {
			return err
		}
	}
	return nil
}
