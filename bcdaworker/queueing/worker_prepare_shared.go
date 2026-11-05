package queueing

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	workerRepo "github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/log"
	mw "github.com/CMSgov/bcda-app/middleware"
	"github.com/ccoveille/go-safecast"
	pgxv5 "github.com/jackc/pgx/v5"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type PrepareSharedJobWorker struct {
	river.WorkerDefaults[worker_types.PrepareSharedJobArgs]
	db   *sql.DB
	pool *pgxv5Pool.Pool
}

func (w *PrepareSharedJobWorker) Work(ctx context.Context, rjob *river.Job[worker_types.PrepareSharedJobArgs]) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
	ctx = context.WithValue(ctx, mw.CtxTransactionKey, rjob.Args.TransactionID)
	logger := log.GetCtxLogger(ctx)

	exports, err := w.prepareSharedExportJobs(ctx, rjob.Args)
	if err != nil {
		logger.Errorf("failed to add jobs to the main queue: %s", err)
		return err
	}

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if rollBackErr := tx.Rollback(ctx); rollBackErr != nil {
				logger.Errorf("Failed to rollback pgx transaction: %s", rollBackErr.Error())
			}
		}
	}()

	client := river.ClientFromContext[pgxv5.Tx](ctx)
	q := riverEnqueuer{pool: w.pool, Client: client}
	err = w.queueExportJobs(ctx, tx, q, rjob.Args, exports)
	if err != nil {
		logger.Errorf("failed to add jobs to the main queue: %s", err)

		repo := workerRepo.NewRepository(w.db)
		dbErr := repo.UpdateJobStatus(ctx, rjob.Args.Job.ID, models.JobStatusFailed)
		if dbErr != nil {
			logger.Errorf("failed to update job status: %s", dbErr)
		}

		return err
	}

	err = tx.Commit(ctx)

	return err
}

// prepareExportJobs builds a list of subjobs to be processed based on the requested queries
func (p *PrepareSharedJobWorker) prepareSharedExportJobs(ctx context.Context, args worker_types.PrepareSharedJobArgs) ([]*worker_types.JobEnqueueArgs, error) {
	var err error
	var exports []*worker_types.JobEnqueueArgs
	repo := postgres.NewRepository(p.db)

	defer func() {
		if err != nil {
			args.Job.Status = models.JobStatusFailed
		}
		dberr := repo.UpdateJob(ctx, args.Job)
		if dberr != nil {
			err = fmt.Errorf("%w: %w", err, dberr)
		}
	}()

	args.Job.TransactionTime, err = getBundleLastUpdated(args)

	exports, err = p.chunkIntoSubJobs(ctx, args)
	if err != nil {
		return exports, err
	}

	args.Job.JobCount = len(exports)
	totalMBIs := 0
	for _, query := range args.Queries {
		totalMBIs += len(query.MBIs)
	}
	// TODO 10590 validate whether this is accurate (are MBIs between queries unique?)
	args.Job.BenesAttributedToACO = totalMBIs

	return exports, err
}

func (p *PrepareSharedJobWorker) chunkIntoSubJobs(ctx context.Context, args worker_types.PrepareSharedJobArgs) ([]*worker_types.JobEnqueueArgs, error) {
	var jobs []*worker_types.JobEnqueueArgs

	for _, query := range args.Queries {
		for _, resourceType := range query.ResourceTypes {
			maxBeneficiaries, err := service.GetMaxBeneCount(resourceType)
			if err != nil {
				return nil, fmt.Errorf("failed to get max beneficiaries for resource type %s: %w", resourceType, err)
			}

			chunks := service.ChunkByIDs(query.MBIs, maxBeneficiaries)
			for _, beneIDs := range chunks {
				// TODO 10590: validate this is correct logic (resource type determines data type, we wont use the passed in DataTypes from the payload)
				// TODO verify that we dont actually use DataType downstream anywhere?
				dataType := service.GetDataType(resourceType)

				id, err := safecast.ToInt(args.Job.ID)
				if err != nil {
					return jobs, err
				}

				enqueueArgs := worker_types.JobEnqueueArgs{
					ID:              id,
					CMSID:           args.CMSID,
					ACOID:           args.Job.ACOID.String(),
					BeneficiaryIDs:  beneIDs, // in the case of shared export these are MBIs
					ResourceType:    resourceType,
					Since:           query.ParsedQuery.Parameters.Since,
					TypeFilter:      query.ParsedQuery.Parameters.TypeFilter,
					TransactionID:   ctx.Value(mw.CtxTransactionKey).(string),
					TransactionTime: args.CreationTime,
					BBBasePath:      args.BFDPath,
					DataType:        dataType,
				}

				if dataType == constants.Adjudicated {
					enqueueArgs.TransactionTime = args.Job.TransactionTime
				}

				// TODO 10590 verify that we dont need to set ClaimsWindow in context of shared export and v3

				jobs = append(jobs, &enqueueArgs)
			}
		}
	}

	return jobs, nil
}

func getBundleLastUpdated(args worker_types.PrepareSharedJobArgs) (time.Time, error) {
	client, err := client.NewBlueButtonClient(client.NewConfig(constants.BFDV3Path))
	if err != nil {
		return time.Time{}, err
	}

	jobData := worker_types.JobEnqueueArgs{
		BBBasePath:      args.BFDPath,
		Since:           args.Queries[0].ParsedQuery.Parameters.Since,
		TransactionTime: time.Now(),
	}

	args.Job.TransactionTime, err = service.GetBundleLastUpdated(client, jobData)
	if err != nil {
		return time.Time{}, err
	}

	return args.Job.TransactionTime, nil
}

func (p *PrepareSharedJobWorker) queueExportJobs(ctx context.Context, tx pgxv5.Tx, q Enqueuer, args worker_types.PrepareSharedJobArgs, exports []*worker_types.JobEnqueueArgs) error {
	jobPriority := service.GetJobPriority(args.CMSID)

	for _, j := range exports {
		if err := q.AddJob(ctx, tx, *j, int(jobPriority)); err != nil {
			return err
		}
	}

	return nil
}
