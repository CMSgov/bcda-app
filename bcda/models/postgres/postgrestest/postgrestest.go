// Package postgrestest provides CRUD utilities for the postgres database.
package postgrestest

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/huandu/go-sqlbuilder"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	sqlFlavor = sqlbuilder.PostgreSQL
)

func CreateACO(t *testing.T, db *sql.DB, aco models.ACO) {
	ib := sqlFlavor.NewInsertBuilder().InsertInto("acos")
	ib.Cols("uuid", "cms_id", "name").Values(aco.UUID, aco.CMSID, aco.Name)
	query, args := ib.Build()
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}

func GetACOByUUID(t *testing.T, db *sql.DB, uuid uuid.UUID) models.ACO {
	sb := sqlFlavor.NewSelectBuilder().Select("uuid", "cms_id", "name").
		From("acos")
	sb.Where(sb.Equal("uuid", uuid)).Limit(1)
	query, args := sb.Build()

	var aco models.ACO
	err := db.QueryRow(query, args...).Scan(&aco.UUID, &aco.CMSID, &aco.Name)
	assert.NoError(t, err)

	return aco
}

func DeleteACO(t *testing.T, db *sql.DB, acoID uuid.UUID) {
	builder := sqlFlavor.NewDeleteBuilder().DeleteFrom("acos")
	builder.Where(builder.Equal("uuid", acoID))

	query, args := builder.Build()
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}

func CreateJobs(t *testing.T, db *sql.DB, jobs ...*models.Job) {
	r := postgres.NewRepository(db)
	for i, j := range jobs {
		id, err := r.CreateJob(context.Background(), *j)
		assert.NoError(t, err)

		createdJob, err := r.GetJobByID(context.Background(), id)
		assert.NoError(t, err)
		fmt.Printf("createdJob - %d %s %s\n", createdJob.ID, createdJob.CreatedAt.String(), createdJob.UpdatedAt.String())

		jobs[i].ID, jobs[i].CreatedAt, jobs[i].UpdatedAt = id, createdJob.CreatedAt, createdJob.UpdatedAt
	}
}

func UpdateJob(t *testing.T, db *sql.DB, j models.Job) {
	r := postgres.NewRepository(db)
	assert.NoError(t, r.UpdateJob(context.Background(), j))

	// Our tests may need to set a specific created_at, updated_at that does not
	// match up with NOW(), we'll update these values out of band if that's the case.
	ub := sqlFlavor.NewUpdateBuilder().Update("jobs")
	if !j.UpdatedAt.IsZero() {
		ub.SetMore(ub.Assign("updated_at", j.UpdatedAt))
	}
	if !j.CreatedAt.IsZero() {
		ub.SetMore(ub.Assign("created_at", j.CreatedAt))
	}
	ub.Where(ub.Equal("id", j.ID))

	query, args := ub.Build()

	fmt.Printf("QUERY: %s ARGS %v", query, args)
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}

func GetJobsByACOID(t *testing.T, db *sql.DB, acoID uuid.UUID) []*models.Job {
	r := postgres.NewRepository(db)
	jobs, err := r.GetJobs(context.Background(), acoID, models.AllJobStatuses...)
	assert.NoError(t, err)
	return jobs
}

func GetJobByID(t *testing.T, db *sql.DB, jobID uint) *models.Job {
	r := postgres.NewRepository(db)
	j, err := r.GetJobByID(context.Background(), jobID)
	assert.NoError(t, err)

	return j
}

func DeleteJobsByACOID(t *testing.T, db *sql.DB, acoID uuid.UUID) {
	builder := sqlFlavor.NewDeleteBuilder().DeleteFrom("jobs")
	builder.Where(builder.Equal("aco_id", acoID))

	query, args := builder.Build()
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}

func DeleteJobByID(t *testing.T, db *sql.DB, jobID uint) {
	builder := sqlFlavor.NewDeleteBuilder().DeleteFrom("jobs")
	builder.Where(builder.Equal("id", jobID))

	query, args := builder.Build()
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}

func CreateJobKeys(t *testing.T, db *sql.DB, jobKeys ...models.JobKey) {
	r := postgres.NewRepository(db)
	err := r.CreateJobKeys(context.Background(), jobKeys...)
	assert.NoError(t, err)
}
