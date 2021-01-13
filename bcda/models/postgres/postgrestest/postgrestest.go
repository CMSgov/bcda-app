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
	ib.Cols("uuid", "cms_id", "name", "blacklisted").Values(aco.UUID, aco.CMSID, aco.Name, aco.Blacklisted)
	query, args := ib.Build()
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}

func GetACOByUUID(t *testing.T, db *sql.DB, uuid uuid.UUID) models.ACO {
	sb := sqlFlavor.NewSelectBuilder().Select("uuid", "cms_id", "name", "blacklisted").
		From("acos")
	sb.Where(sb.Equal("uuid", uuid)).Limit(1)
	query, args := sb.Build()

	var aco models.ACO
	err := db.QueryRow(query, args...).Scan(&aco.UUID, &aco.CMSID, &aco.Name, &aco.Blacklisted)
	assert.NoError(t, err)

	return aco
}

func GetACOByCMSID(t *testing.T, db *sql.DB, cmsID string) models.ACO {
	sb := sqlFlavor.NewSelectBuilder().Select("uuid", "cms_id", "name").
		From("acos")
	sb.Where(sb.Equal("cms_id", cmsID)).Limit(1)
	query, args := sb.Build()

	var aco models.ACO
	err := db.QueryRow(query, args...).Scan(&aco.UUID, &aco.CMSID, &aco.Name)
	assert.NoError(t, err)

	return aco
}

func UpdateACO(t *testing.T, db *sql.DB, aco models.ACO) {
	ub := sqlFlavor.NewUpdateBuilder().Update("acos")
	ub.Set(
		ub.Assign("cms_id", aco.CMSID),
		ub.Assign("name", aco.Name),
		ub.Assign("client_id", aco.ClientID),
		ub.Assign("group_id", aco.GroupID),
		ub.Assign("system_id", aco.SystemID),
		ub.Assign("alpha_secret", aco.AlphaSecret),
		ub.Assign("public_key", aco.PublicKey),
		ub.Assign("blacklisted", aco.Blacklisted),
	).Where(ub.Equal("uuid", aco.UUID))

	query, args := ub.Build()
	result, err := db.Exec(query, args...)
	assert.NoError(t, err)

	count, err := result.RowsAffected()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, count)
}

func DeleteACO(t *testing.T, db *sql.DB, acoID uuid.UUID) {
	builder := sqlFlavor.NewDeleteBuilder().DeleteFrom("acos")
	builder.Where(builder.Equal("uuid", acoID))

	query, args := builder.Build()
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}

// DeleteCCLFFilesByCMSID deletes all CCLFFile associated with a particular ACO represented by cmsID
// Since (as of 2021-01-13), we have foreign key ties to this table, we'll
// also delete any relational ties (e.g. CCLFBeneficiary)
func DeleteCCLFFilesByCMSID(t *testing.T, db *sql.DB, cmsID string) {
	fileIDs, err := getFileIDsForCMSID(db, cmsID)
	assert.NoError(t, err)

	if len(fileIDs) > 0 {
		beneDelete := sqlFlavor.NewDeleteBuilder().DeleteFrom("cclf_beneficiaries")
		beneDelete = beneDelete.Where(beneDelete.In("file_id", fileIDs...))

		query, args := beneDelete.Build()
		_, err = db.Exec(query, args...)
		assert.NoError(t, err)
	}

	cclfFileDelete := sqlFlavor.NewDeleteBuilder().DeleteFrom("cclf_files")
	cclfFileDelete.Where(cclfFileDelete.Equal("aco_cms_id", cmsID))

	query, args := cclfFileDelete.Build()
	_, err = db.Exec(query, args...)
	assert.NoError(t, err)
}

func CreateJobs(t *testing.T, db *sql.DB, jobs ...*models.Job) {
	r := postgres.NewRepository(db)
	for i, j := range jobs {
		id, err := r.CreateJob(context.Background(), *j)
		assert.NoError(t, err)

		// Callers may want to set a custom createdAt or updatedAt
		if !j.CreatedAt.IsZero() || !j.UpdatedAt.IsZero() {
			j.ID = id
			UpdateJob(t, db, *j)
		}

		createdJob, err := r.GetJobByID(context.Background(), id)
		assert.NoError(t, err)

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

func GetSuppressionFileByName(t *testing.T, db *sql.DB, names ...string) []models.SuppressionFile {
	nameArgs := make([]interface{}, len(names))
	for i, name := range names {
		nameArgs[i] = name
	}

	sb := sqlFlavor.NewSelectBuilder().Select("id", "name", "timestamp", "import_status").From("suppression_files")
	sb.Where(sb.In("name", nameArgs...))

	query, args := sb.Build()
	rows, err := db.Query(query, args...)
	assert.NoError(t, err)

	defer rows.Close()

	var files []models.SuppressionFile
	for rows.Next() {
		var sf models.SuppressionFile
		err = rows.Scan(&sf.ID, &sf.Name, &sf.Timestamp, &sf.ImportStatus)
		assert.NoError(t, err)
		files = append(files, sf)
	}

	assert.NoError(t, rows.Err())

	return files
}

func DeleteSuppressionFileByID(t *testing.T, db *sql.DB, id uint) {
	delete := sqlFlavor.NewDeleteBuilder().DeleteFrom("suppression_files")
	delete.Where(delete.Equal("id", id))

	query, args := delete.Build()
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}

func getFileIDsForCMSID(db *sql.DB, cmsID string) ([]interface{}, error) {
	sb := sqlFlavor.NewSelectBuilder().Select("id").From("cclf_files")
	sb.Where(sb.Equal("aco_cms_id", cmsID))
	query, args := sb.Build()
	var ids []interface{}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var id uint
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}
