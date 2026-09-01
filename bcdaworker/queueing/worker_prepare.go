/*
Prepare Worker takes all of the arguments of a bulk export request to BCDA API and asynchronously prepares and
creates (enqueues?) all of the subjobs needed for the requests bulk export main job.
*/

package queueing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"slices"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	fhirModels "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	m "github.com/CMSgov/bcda-app/middleware"
	"github.com/ccoveille/go-safecast"
	pgxv5 "github.com/jackc/pgx/v5"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
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
	pool     *pgxv5Pool.Pool
}

func NewPrepareJobWorker(db *sql.DB, pool *pgxv5Pool.Pool) (*PrepareJobWorker, error) {

	logger := log.Worker
	client.SetLogger(logger)

	cfg, err := service.LoadConfig()
	if err != nil {
		logger.Fatalf("failed to load service config. Err: %v", err)
	}
	if len(cfg.ACOConfigs) == 0 {
		logger.Fatalf("no ACO configs found, these are required for downstream processing")
	}

	repository := postgres.NewRepository(db)
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

	return &PrepareJobWorker{svc: svc, v1Client: v1, v2Client: v2, v3Client: v3, r: repository, pool: pool}, nil

}

func (w *PrepareJobWorker) Work(ctx context.Context, rjob *river.Job[worker_types.PrepareJobArgs]) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			ctx = log.NewStructuredLoggerEntry(log.Worker, ctx)
			ctx = context.WithValue(ctx, m.CtxTransactionKey, rjob.Args.TransactionID)
			logger := log.GetCtxLogger(ctx)

			exports, since, err := w.prepareExportJobs(ctx, rjob.Args)
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
			err = w.queueExportJobs(ctx, tx, q, rjob.Args, exports, since)
			if err != nil {
				// TODO update job in jobs table as failed
				logger.Errorf("failed to add jobs to the main queue: %s", err)
				return err
			}

			err = tx.Commit(ctx)
			if err != nil {
				logger.Errorf("prepare job tx failed for job id: %d, err: %v", rjob.Args.Job.ID, err)
				return err
			}

			if defaultSystemTypeWarningNeeded(rjob.Args.Job.RequestURL, rjob.Args.BFDPath, rjob.Args.ResourceTypes) {
				err = handleDefaultSystemTypeWarning(ctx, w.pool, rjob)
				if err != nil {
					// dont fail main job process because of this warning, to be re-considered if need be
					logger.Errorf("failed to check if default system type warning needed for job id: %d, err: %v", rjob.Args.Job.ID, err)
				}
			}

			return nil
		}
	}
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

	exports, benesAttributed, err := p.svc.GetQueJobs(ctx, args)
	if err != nil {
		logger.Error(err)
		if ok := errors.As(err, &service.CCLFNotFoundError{}); ok {
			return exports, args.Since, err
		} else {
			return exports, args.Since, err
		}
	}

	args.Job.JobCount = len(exports)
	args.Job.BenesAttributedToACO = benesAttributed

	return exports, args.Since, err
}

// GetBundleLastUpdated requests a fake patient in order to acquire the bundle's lastUpdated metadata.
func (p *PrepareJobWorker) GetBundleLastUpdated(basepath string, jobData worker_types.JobEnqueueArgs) (time.Time, error) {
	var (
		b   *fhirModels.Bundle
		err error
	)

	switch basepath {
	case constants.BFDV1Path:
		b, err = p.v1Client.GetPatient(jobData, "0")
	case constants.BFDV2Path:
		b, err = p.v2Client.GetPatient(jobData, "0")
	case constants.BFDV3Path:
		b, err = p.v3Client.GetPatient(jobData, "0")
	default:
		return time.Time{}, fmt.Errorf("unsupported BFD base path: %s", basepath)
	}

	if err != nil {
		return time.Time{}, err
	}

	// Safeguard: If BFD lower environments return 1970/epoch or an unpopulated timestamp, fallback to request transaction time
	if b == nil || b.Meta.LastUpdated.IsZero() || b.Meta.LastUpdated.Year() < 2000 {
		return jobData.TransactionTime, nil
	}

	return b.Meta.LastUpdated, nil
}

func (p *PrepareJobWorker) queueExportJobs(ctx context.Context, tx pgxv5.Tx, q Enqueuer, args worker_types.PrepareJobArgs, exports []*worker_types.JobEnqueueArgs, since time.Time) error {
	for _, j := range exports {
		sinceParam := !since.IsZero() || args.RequestType == constants.RetrieveNewBeneHistData
		jobPriority := p.svc.GetJobPriority(args.CMSID, j.ResourceType, sinceParam)

		if err := q.AddJob(ctx, tx, *j, int(jobPriority)); err != nil {
			return err
		}
	}
	return nil
}

// handleDefaultSystemTypeWarning checks if a default system type warning is needed and appends it to the warnings and info file.
func handleDefaultSystemTypeWarning(ctx context.Context, pool *pgxv5Pool.Pool, rjob *river.Job[worker_types.PrepareJobArgs]) error {
	pgxRepo := postgres.NewPgxRepositoryWithPool(pool)

	err := service.SetupWarningsAndInfoFile(ctx, pgxRepo, rjob.Args.Job.ID)
	if err != nil {
		return err
	}

	bytes, err := json.Marshal(service.WarningDefaultSystemType)
	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s/%d/%s", conf.GetEnv("FHIR_PAYLOAD_DIR"), rjob.Args.Job.ID, constants.WarningsAndInfoFileName)
	bytes = append(bytes, []byte("\n")...)                                        // add newline to end of OpOutcome json
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600) // #nosec G304
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(bytes)
	return err
}

// defaultSystemTypeWarningNeeded checks if a default system type warning is needed based on various request parameters.
// The warning is needed when 1) its a v3 request 2) for EOB resources and 3) the request url did NOT specify a system type (wherein we pass in default system types of NCH and DDPS).
// when needed we are adding a default System-Type typeFilter onto BFD requests and therefore a warning is needed.
// we dont care about error returns because we dont want this to block the job from being processed as well as
// these errors can sometimes mean that we need to add the warning.
func defaultSystemTypeWarningNeeded(requestURL string, version string, resourceTypes []string) bool {
	if version != constants.BFDV3Path {
		return false
	}

	if !slices.Contains(resourceTypes, "ExplanationOfBenefit") {
		return false
	}

	URL, err := url.Parse(requestURL)
	if err != nil {
		return true
	}

	params, ok := URL.Query()["_typeFilter"]
	if !ok {
		return true
	}

	typeFilterParams, err := middleware.GetTypeFilterParams(params)
	if err != nil {
		return true
	}

	if middleware.HasSharedSystemTag(typeFilterParams) {
		return false
	}

	return true
}
