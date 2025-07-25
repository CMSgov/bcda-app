// Package postgrestest provides CRUD utilities for the postgres database.
// These utilities allow the caller to modify the database in ways that we
// wouldn't want to permit in the main code path.
// To protect against usage in non-test code, all methods should accept
// a *testing.T struct.
package postgrestest

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/optout"
	"github.com/huandu/go-sqlbuilder"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	sqlFlavor = sqlbuilder.PostgreSQL
)

func CreateACO(t *testing.T, db *sql.DB, aco models.ACO) {
	r := postgres.NewRepository(db)
	assert.NoError(t, r.CreateACO(context.Background(), aco))
}

func GetACOByUUID(t *testing.T, db *sql.DB, uuid uuid.UUID) models.ACO {
	r := postgres.NewRepository(db)
	aco, err := r.GetACOByUUID(context.Background(), uuid)
	assert.NoError(t, err)
	return *aco
}

func GetACOByCMSID(t *testing.T, db *sql.DB, cmsID string) models.ACO {
	r := postgres.NewRepository(db)
	aco, err := r.GetACOByCMSID(context.Background(), cmsID)
	assert.NoError(t, err)
	return *aco
}

func UpdateACO(t *testing.T, db *sql.DB, aco models.ACO) {
	r := postgres.NewRepository(db)
	// This should capture ALL fields present in the ACO model
	fieldsAndValues := map[string]interface{}{
		"cms_id":              aco.CMSID,
		"name":                aco.Name,
		"client_id":           aco.ClientID,
		"group_id":            aco.GroupID,
		"system_id":           aco.SystemID,
		"termination_details": aco.TerminationDetails,
	}
	assert.NoError(t, r.UpdateACO(context.Background(), aco.UUID, fieldsAndValues))
}

// DeleteACO also removes data from any foreign key relations (jobs) before deleting the ACO.
func DeleteACO(t *testing.T, db *sql.DB, acoID uuid.UUID) {
	DeleteJobsByACOID(t, db, acoID)

	builder := sqlFlavor.NewDeleteBuilder().DeleteFrom("acos")
	builder.Where(builder.Equal("uuid", acoID))

	query, args := builder.Build()
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}

func CreateCCLFBeneficiary(t *testing.T, db *sql.DB, bene *models.CCLFBeneficiary) {
	ib := sqlFlavor.NewInsertBuilder().InsertInto("cclf_beneficiaries")
	ib.Cols("file_id", "mbi", "blue_button_id").
		Values(bene.FileID, bene.MBI, bene.BlueButtonID)
	query, args := ib.Build()
	// Append the RETURNING id to retrieve the auto-generated ID value associated with the bene
	query = fmt.Sprintf("%s RETURNING id", query)

	err := db.QueryRow(query, args...).Scan(&bene.ID)
	assert.NoError(t, err)
}
func CreateCCLFFile(t *testing.T, db *sql.DB, cclfFile *models.CCLFFile) {
	r := postgres.NewRepository(db)
	var err error
	cclfFile.ID, err = r.CreateCCLFFile(context.Background(), *cclfFile)
	assert.NoError(t, err)
}

func GetCCLFFilesByCMSID(t *testing.T, db *sql.DB, cmsID string) []models.CCLFFile {
	cclfFiles, err := getCCLFFiles(db, "aco_cms_id", cmsID)
	assert.NoError(t, err)
	return cclfFiles
}

func GetCCLFFilesByName(t *testing.T, db *sql.DB, name string) []models.CCLFFile {
	cclfFiles, err := getCCLFFiles(db, "name", name)
	assert.NoError(t, err)
	return cclfFiles
}

func GetLatestCCLFFileByCMSIDAndType(t *testing.T, db *sql.DB, cmsID string, fileType models.CCLFFileType) models.CCLFFile {
	sb := sqlFlavor.NewSelectBuilder().
		Select("id", "cclf_num", "name", "aco_cms_id", "timestamp", "performance_year", "import_status", "type").
		From("cclf_files")
	sb.Where(
		sb.Equal("aco_cms_id", cmsID),
		sb.Equal("type", fileType),
	)
	sb.OrderBy("timestamp").Desc().Limit(1)

	query, args := sb.Build()
	row := db.QueryRow(query, args...)

	var cclfFile models.CCLFFile
	err := row.Scan(&cclfFile.ID, &cclfFile.CCLFNum, &cclfFile.Name,
		&cclfFile.ACOCMSID, &cclfFile.Timestamp, &cclfFile.PerformanceYear,
		&cclfFile.ImportStatus, &cclfFile.Type)
	assert.NoError(t, err)

	return cclfFile
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
	ib := sqlFlavor.NewInsertBuilder().InsertInto("job_keys")
	ib.Cols("job_id", "file_name", "resource_type")
	for _, key := range jobKeys {
		ib.Values(key.JobID, key.FileName, key.ResourceType)
	}

	query, args := ib.Build()
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}

func DeleteJobKeysByJobIDs(t *testing.T, db *sql.DB, jobIDs ...uint) {
	ids := make([]interface{}, len(jobIDs))
	for i, id := range jobIDs {
		ids[i] = id
	}

	builder := sqlFlavor.NewDeleteBuilder().DeleteFrom("job_keys")
	builder.Where(builder.In("job_id", ids...))
	query, args := builder.Build()
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}

func GetSuppressionFileByName(t *testing.T, db *sql.DB, names ...string) []optout.OptOutFile {
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

	var files []optout.OptOutFile
	for rows.Next() {
		var sf optout.OptOutFile
		err = rows.Scan(&sf.ID, &sf.Name, &sf.Timestamp, &sf.ImportStatus)
		assert.NoError(t, err)
		files = append(files, sf)
	}

	assert.NoError(t, rows.Err())

	return files
}

// DeleteSuppressionFileByID deletes the suppresion file associated with the given ID.
// Since (as of 2021-01-13), we have foreign key ties to this table, we'll
// also delete any relational ties (e.g. Suppressions)
func DeleteSuppressionFileByID(t *testing.T, db *sql.DB, id uint) {
	deleteSuppression := sqlFlavor.NewDeleteBuilder().DeleteFrom("suppressions")
	deleteSuppression.Where(deleteSuppression.Equal("file_id", id))
	query, args := deleteSuppression.Build()
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)

	delete := sqlFlavor.NewDeleteBuilder().DeleteFrom("suppression_files")
	delete.Where(delete.Equal("id", id))
	query, args = delete.Build()
	_, err = db.Exec(query, args...)
	assert.NoError(t, err)
}

func GetSuppressionsByFileID(t *testing.T, db *sql.DB, fileID uint) []optout.OptOutRecord {
	sb := sqlFlavor.NewSelectBuilder().Select("id", "file_id", "mbi", "source_code", "effective_date", "preference_indicator",
		"samhsa_source_code", "samhsa_effective_date", "samhsa_preference_indicator",
		"beneficiary_link_key", "aco_cms_id").From("suppressions")
	sb.Where(sb.Equal("file_id", fileID))
	sb.OrderBy("id")

	query, args := sb.Build()
	rows, err := db.Query(query, args...)
	assert.NoError(t, err)
	defer rows.Close()

	var suppressions []optout.OptOutRecord
	for rows.Next() {
		var s optout.OptOutRecord
		err = rows.Scan(&s.ID, &s.FileID, &s.MBI, &s.SourceCode, &s.EffectiveDt, &s.PrefIndicator,
			&s.SAMHSASourceCode, &s.SAMHSAEffectiveDt, &s.SAMHSAPrefIndicator,
			&s.BeneficiaryLinkKey, &s.ACOCMSID)
		assert.NoError(t, err)
		suppressions = append(suppressions, s)
	}
	assert.NoError(t, rows.Err())

	return suppressions
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

func getCCLFFiles(db *sql.DB, field, value string) ([]models.CCLFFile, error) {
	sb := sqlFlavor.NewSelectBuilder().
		Select("id", "cclf_num", "name", "aco_cms_id", "timestamp", "performance_year", "import_status", "type").
		From("cclf_files")
	sb.Where(sb.Equal(field, value))

	query, args := sb.Build()
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var cclfFiles []models.CCLFFile
	for rows.Next() {
		var cclfFile models.CCLFFile
		if err := rows.Scan(&cclfFile.ID, &cclfFile.CCLFNum, &cclfFile.Name,
			&cclfFile.ACOCMSID, &cclfFile.Timestamp, &cclfFile.PerformanceYear,
			&cclfFile.ImportStatus, &cclfFile.Type); err != nil {
			return nil, err
		}
		cclfFiles = append(cclfFiles, cclfFile)
	}

	if rows.Err() != nil {
		return nil, err
	}

	return cclfFiles, nil
}

func GetJobKey(db *sql.DB, jobID int) ([]models.JobKey, error) {
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder().Select("id", "job_id", "file_name", "resource_type").From("job_keys")
	sb.Where(sb.Equal("job_id", jobID))
	query, args := sb.Build()
	fmt.Println(query)
	fmt.Println(args)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var jobKeys []models.JobKey
	for rows.Next() {
		var jobKey models.JobKey
		if err := rows.Scan(&jobKey.ID, &jobKey.JobID, &jobKey.FileName,
			&jobKey.ResourceType); err != nil {
			return nil, err
		}
		jobKeys = append(jobKeys, jobKey)
	}
	return jobKeys, err
}

func GetCCLFBeneficiaries(db *sql.DB, file_id int) ([]string, error) {
	var mbis []string
	sb := sqlFlavor.NewSelectBuilder().Select("mbi").From("cclf_beneficiaries")
	sb.Where(sb.Equal("file_id", file_id))
	query, args := sb.Build()
	rows, err := db.Query(query, args...)
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
