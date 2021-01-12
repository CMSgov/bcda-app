// Package postgrestest provides CRUD utilities for the postgres database.
package postgrestest

import (
	"database/sql"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/models"
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

func CreateJobs(t *testing.T, db *sql.DB, jobs ...models.Job) {
	ib := sqlFlavor.NewInsertBuilder().InsertInto("jobs")
	ib.Cols("aco_id", "request_url", "status")

	for _, j := range jobs {
		ib.Values(j.ACOID, j.RequestURL, j.Status)
	}

	query, args := ib.Build()
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}

func DeleteJobsByACOID(t *testing.T, db *sql.DB, acoID uuid.UUID) {
	builder := sqlFlavor.NewDeleteBuilder().DeleteFrom("jobs")
	builder.Where(builder.Equal("aco_id", acoID))

	query, args := builder.Build()
	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}
