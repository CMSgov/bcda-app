package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/models"
)

type queryable interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type executable interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

const (
	sqlFlavor = sqlbuilder.PostgreSQL
)

// Ensure Repository satisfies the interface
var _ models.Repository = &Repository{}

type Repository struct {
	queryable
	executable
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db, db}
}

func NewRepositoryTx(tx *sql.Tx) *Repository {
	return &Repository{tx, tx}
}

func (r *Repository) GetLatestCCLFFile(ctx context.Context, cmsID string, cclfNum int, importStatus string, lowerBound, upperBound time.Time, fileType models.CCLFFileType) (*models.CCLFFile, error) {

	const (
		queryNoTime     = "aco_cms_id = ? AND cclf_num = ? AND import_status = ? AND type = ?"
		queryLower      = queryNoTime + " AND timestamp >= ?"
		queryUpper      = queryNoTime + " AND timestamp <= ?"
		queryLowerUpper = queryNoTime + " AND timestamp >= ? AND timestamp <= ?"
	)

	sb := sqlFlavor.NewSelectBuilder()
	sb.Select("id", "name", "timestamp", "performance_year")
	sb.From("cclf_files")
	sb.Where(
		sb.Equal("aco_cms_id", cmsID),
		sb.Equal("cclf_num", cclfNum),
		sb.Equal("import_status", importStatus),
		sb.Equal("type", fileType),
	)

	cclfFile := models.CCLFFile{
		ACOCMSID:     cmsID,
		CCLFNum:      cclfNum,
		ImportStatus: importStatus,
		Type:         fileType,
	}

	if !lowerBound.IsZero() && upperBound.IsZero() {
		sb.Where(sb.GreaterEqualThan("timestamp", lowerBound))
	} else if lowerBound.IsZero() && !upperBound.IsZero() {
		sb.Where(sb.LessEqualThan("timestamp", upperBound))
	} else if !lowerBound.IsZero() && !upperBound.IsZero() {
		sb.Where(
			sb.GreaterEqualThan("timestamp", lowerBound),
			sb.LessEqualThan("timestamp", upperBound),
		)
	}
	sb.OrderBy("timestamp").Desc().Limit(1)

	query, args := sb.Build()
	row := r.QueryRowContext(ctx, query, args...)
	if err := row.Scan(&cclfFile.ID, &cclfFile.Name, &cclfFile.Timestamp, &cclfFile.PerformanceYear); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &cclfFile, nil
}

func (r *Repository) GetCCLFBeneficiaryMBIs(ctx context.Context, cclfFileID uint) ([]string, error) {
	var mbis []string

	sb := sqlFlavor.NewSelectBuilder().Select("mbi").From("cclf_beneficiaries")
	sb.Where(sb.Equal("file_id", cclfFileID))

	query, args := sb.Build()
	rows, err := r.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mbi string
		if err = rows.Scan(&mbi); err != nil {
			return nil, err
		}
		mbis = append(mbis, mbi)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return mbis, nil
}

func (r *Repository) GetCCLFBeneficiaries(ctx context.Context, cclfFileID uint, ignoredMBIs []string) ([]*models.CCLFBeneficiary, error) {
	var beneficiaries []*models.CCLFBeneficiary

	// Subquery to deal with duplicate MBIs found within a single CCLF file.
	// NOTE: We no longer have duplicate MBIs after this PR: https://github.com/CMSgov/bcda-app/pull/583
	// We have to remove duplicates on older files, but once that's done, we can remove the subquery
	// and query for the benes by file_id directly.
	subSB := sqlFlavor.NewSelectBuilder()
	subSB.Select("MAX(id)").From("cclf_beneficiaries").Where(
		subSB.Equal("file_id", cclfFileID),
	).GroupBy("mbi")

	sb := sqlFlavor.NewSelectBuilder()
	sb.Select("id", "file_id", "mbi", "blue_button_id")
	sb.From("cclf_beneficiaries").Where(sb.In("id", subSB))

	if len(ignoredMBIs) != 0 {
		ignored := make([]interface{}, len(ignoredMBIs))
		for i, v := range ignoredMBIs {
			ignored[i] = v
		}
		sb.Where(sb.NotIn("mbi", ignored...))
	}

	query, args := sb.Build()
	rows, err := r.QueryContext(ctx, query, args...)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var bene models.CCLFBeneficiary
		if err := rows.Scan(&bene.ID, &bene.FileID, &bene.MBI, &bene.BlueButtonID); err != nil {
			return nil, err
		}
		beneficiaries = append(beneficiaries, &bene)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return beneficiaries, nil
}

func (r *Repository) GetSuppressedMBIs(ctx context.Context, lookbackDays int) ([]string, error) {
	var suppressedMBIs []string

	subSB := sqlFlavor.NewSelectBuilder()
	subSB.Select("mbi", "MAX(effective_date) as max_date").From("suppressions")
	subSB.Where(
		subSB.Between("effective_date", sqlbuilder.Raw(fmt.Sprintf("NOW() - interval '%d days'", lookbackDays)), sqlbuilder.Raw("NOW()")),
		subSB.NotEqual("preference_indicator", ""),
	).GroupBy("mbi")

	sb := sqlFlavor.NewSelectBuilder().Distinct().Select("s.mbi")
	sb.From(sb.BuilderAs(subSB, "h")).Join("suppressions s", "s.mbi = h.mbi", "s.effective_date = h.max_date")
	sb.Where(sb.Equal("preference_indicator", "N"))

	query, args := sb.Build()
	rows, err := r.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mbi string
		if err = rows.Scan(&mbi); err != nil {
			return nil, err
		}
		suppressedMBIs = append(suppressedMBIs, mbi)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return suppressedMBIs, nil
}

func (r *Repository) GetJobs(ctx context.Context, acoID uuid.UUID, statues ...models.JobStatus) ([]models.Job, error) {
	s := make([]interface{}, len(statues))
	for i, v := range statues {
		s[i] = v
	}

	sb := sqlFlavor.NewSelectBuilder()
	sb.Select("id", "request_url", "status", "transaction_time", "job_count", "completed_job_count", "created_at", "updated_at")
	sb.From("jobs").Where(
		sb.Equal("aco_id", acoID),
		sb.In("status", s...),
	)

	query, args := sb.Build()
	rows, err := r.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []models.Job
	for rows.Next() {
		j := models.Job{ACOID: acoID}
		if err = rows.Scan(&j.ID, &j.RequestURL, &j.Status, &j.TransactionTime,
			&j.JobCount, &j.CompletedJobCount, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (r *Repository) CreateJob(ctx context.Context, j models.Job) (uint, error) {
	// User raw builder since we need to retrieve the associated ID
	query, args := sqlbuilder.Buildf(`INSERT INTO jobs 
		(aco_id, request_url, status, transaction_time, job_count, completed_job_count) VALUES
		(%s, %s, %s, %s, %s, %s) RETURNING id`,
		j.ACOID, j.RequestURL, j.Status, j.TransactionTime, j.JobCount, j.CompletedJobCount).
		BuildWithFlavor(sqlFlavor)

	var id uint
	if err := r.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (r *Repository) UpdateJob(ctx context.Context, j models.Job) error {
	ub := sqlFlavor.NewUpdateBuilder().Update("jobs")
	ub.Set(
		ub.Assign("aco_id", j.ACOID),
		ub.Assign("request_url", j.RequestURL),
		ub.Assign("status", j.Status),
		ub.Assign("transaction_time", j.TransactionTime),
		ub.Assign("job_count", j.JobCount),
		ub.Assign("completed_job_count", j.CompletedJobCount),
	)
	ub.Where(ub.Equal("id", j.ID))
	query, args := ub.Build()

	res, err := r.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows != 1 {
		return fmt.Errorf("expected to affect 1 row, affected %d", rows)
	}

	return nil
}
