package queueing

import (
	"context"
	"fmt"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/log"
	mw "github.com/CMSgov/bcda-app/middleware"
	"github.com/ccoveille/go-safecast"
	pgxv5 "github.com/jackc/pgx/v5"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type PrepareSharedJobWorker struct {
	river.WorkerDefaults[worker_types.PrepareSharedJobArgs]
	// db   *sql.DB
	pool *pgxv5Pool.Pool
}

// func (w *PrepareSharedJobWorker) Timeout(*river.Job[worker_types.JobEnqueueArgs]) time.Duration {
// 	minutes := utils.GetEnvInt("PROCESS_JOB_TIMEOUT_MINUTES", 30)
// 	return time.Duration(minutes) * time.Minute
// }

func (w *PrepareSharedJobWorker) Work(ctx context.Context, rjob *river.Job[worker_types.PrepareSharedJobArgs]) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
			ctx = context.WithValue(ctx, mw.CtxTransactionKey, rjob.Args.TransactionID)
			logger := log.GetCtxLogger(ctx)

			// exports, since, err := w.prepareExportJobs(ctx, rjob.Args)
			exports, err := w.prepareExportJobs(ctx, rjob.Args)
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
					if err1 := tx.Rollback(ctx); err1 != nil {
						logger.Errorf("Failed to rollback pgx transaction: %s", err1.Error())
					}
				}
			}()

			client := river.ClientFromContext[pgxv5.Tx](ctx)
			q := riverEnqueuer{pool: w.pool, Client: client}
			err = w.queueExportJobs(ctx, tx, q, rjob.Args, exports)
			if err != nil {
				// TODO update job in jobs table as failed
				logger.Errorf("failed to add jobs to the main queue: %s", err)
				return err
			}

			err = tx.Commit(ctx)
			return err
		}
	}
}

// prepareExportJobs builds a list of jobs to be processed based on the parent job.
func (p *PrepareSharedJobWorker) prepareExportJobs(ctx context.Context, args worker_types.PrepareSharedJobArgs) ([]*worker_types.JobEnqueueArgs, error) {
	var err error
	var exports []*worker_types.JobEnqueueArgs
	// logger := log.GetCtxLogger(ctx)
	// repo := postgres.NewPgxRepositoryWithPool(p.pool)

	defer func() {
		if err != nil {
			args.Job.Status = models.JobStatusFailed
		}
		// dberr := repo.UpdateJob(ctx, args.Job) // TODO
		// if dberr != nil {
		// 	err = fmt.Errorf("%w: %w", err, dberr)
		// }
	}()

	// id, err := safecast.ToInt(args.Job.ID)
	// if err != nil {
	// 	return exports, args.Since, err
	// }

	// jobData := worker_types.JobEnqueueArgs{
	// 	ID:    id,
	// 	ACOID: args.PartnerID, // TODO
	// 	Since: args.Since.String(),
	// 	// TypeFilter:      args.TypeFilter,
	// 	TransactionTime: time.Now(),
	// 	CMSID:           args.PartnerID, // TODO
	// }

	// args.Job.TransactionTime, err = p.GetBundleLastUpdated(args.BFDPath, jobData) // we are using jobData.TransactionTime (Now) like we do in v3 requests
	// if err != nil {
	// 	return exports, args.Since, err
	// }

	// exports, benesAttributed, err := p.svc.GetQueJobs(ctx, args)
	exports, err = p.chunkIntoJobs(ctx, args)
	if err != nil {
		return exports, err
	}

	args.Job.JobCount = len(exports)
	args.Job.BenesAttributedToACO = len(args.MBIs)

	return exports, err
}

func (p *PrepareSharedJobWorker) chunkIntoJobs(ctx context.Context, args worker_types.PrepareSharedJobArgs) ([]*worker_types.JobEnqueueArgs, error) {
	var jobs []*worker_types.JobEnqueueArgs

	// pulled from service/service.go func (s *service) createQueueJobs()
	for _, resourceType := range args.ResourceTypes {
		// maxBeneficiaries, err := getMaxBeneCount(resourceType)
		var maxBeneficiaries int
		switch resourceType {
		case "ExplanationOfBenefit":
			maxBeneficiaries = 50
		case "Patient":
			maxBeneficiaries = 5000
		case "Coverage", "Claim", "ClaimResponse":
			maxBeneficiaries = 4000
		default:
			return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
		}

		// chunks, err := chunkBeneficiaryIDs(args.MBIs, maxBeneficiaries)
		if len(args.MBIs) == 0 {
			return nil, fmt.Errorf("no beneficiaries found for resource type: %s", resourceType)
		}

		chunks := make([][]string, 0, (len(args.MBIs)+maxBeneficiaries-1)/maxBeneficiaries)
		currentChunk := make([]string, 0, maxBeneficiaries)

		for _, mbi := range args.MBIs {
			currentChunk = append(currentChunk, fmt.Sprint(mbi))
			if len(currentChunk) < maxBeneficiaries {
				continue
			}

			chunks = append(chunks, currentChunk)
			currentChunk = make([]string, 0, maxBeneficiaries)
		}

		if len(currentChunk) > 0 {
			chunks = append(chunks, currentChunk)
		}

		for _, group := range chunks {
			// resourceJobs, err := s.createJobsForResourceChunk(ctx, args, sinceArg, resourceType, beneficiaryIDs, effectiveDataTypes)
			// if err != nil {
			// 	return nil, err
			// }
			// resource, ok := service.GetClaimType(resourceType)
			// if !ok {
			// 	// This should never be possible, would have returned earlier.
			// 	return nil, errors.New("Invalid resource type: " + resourceType)
			// }

			// tempJobs := make([]*worker_types.JobEnqueueArgs, 0, len(effectiveDataTypes))
			for _, dataType := range args.DataTypes { // constants.PartiallyAdjudicated, constants.Adjudicated
				// if !resource.SupportsClaimType(dataType) {
				// 	continue
				// }

				id, err := safecast.ToInt(args.Job.ID)
				if err != nil {
					return jobs, err
				}

				// enqueueArgs, err := s.buildQueueJobArgs(ctx, args, sinceArg, beneficiaryIDs, resourceType, dataType)
				enqueueArgs := worker_types.JobEnqueueArgs{
					ID: id,
					// ACOID:           args.Job.ACOID.String(),
					ACOID: args.PartnerID, // TODO
					CMSID: args.PartnerID, // TODO
					// CMSID:           args.Job.CMSID,
					BeneficiaryIDs: group,
					ResourceType:   resourceType,
					Since:          args.Since.String(),
					// TypeFilter:      args.TypeFilter,
					TransactionID: ctx.Value(mw.CtxTransactionKey).(string),
					// TransactionTime: getQueueJobTransactionTime(args, dataType),
					TransactionTime: time.Now(),
					BBBasePath:      args.BFDPath,
					DataType:        dataType,
				}

				// if !s.setClaimsDate(&enqueueArgs, args) {
				// 	return nil, &bcdaerrors.InvalidACOConfigError{CMSID: args.CMSID}
				// }
				// 	if err != nil {
				// 		return nil, err
				// 	}
				jobs = append(jobs, &enqueueArgs)
			}
		}
	}

	return jobs, nil
}

func (p *PrepareSharedJobWorker) queueExportJobs(ctx context.Context, tx pgxv5.Tx, q Enqueuer, args worker_types.PrepareSharedJobArgs, exports []*worker_types.JobEnqueueArgs) error {
	for _, j := range exports {
		// sinceParam := !since.IsZero() || args.RequestType == constants.RetrieveNewBeneHistData
		// jobPriority := p.svc.GetJobPriority(args.CMSID, j.ResourceType, sinceParam)

		if err := q.AddJob(ctx, tx, *j, int(4)); err != nil { // TODO priority
			return err
		}
	}
	return nil
}
