package queueing

import (
	"math"
	"math/rand"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/huandu/go-sqlbuilder"
	_ "github.com/jackc/pgx"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

func TestQueEnqueuer(t *testing.T) {
	// Need access to the queue database to ensure we've enqueued the job successfully
	db := database.QueueConnection

	priority := math.MaxInt16
	enqueuer := NewEnqueuer()
	jobArgs := models.JobEnqueueArgs{ID: int(rand.Int31()), ACOID: uuid.New()}
	assert.NoError(t, enqueuer.AddJob(jobArgs, priority))

	// Verify that we've inserted the que_job as expected
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder().Select("COUNT(1)").From("que_jobs")
	sb.Where(sb.Equal("CAST (args ->> 'ID' AS INTEGER)", jobArgs.ID), sb.Equal("args ->> 'ACOID'", jobArgs.ACOID),
		sb.Equal("priority", priority))

	var count int
	query, args := sb.Build()
	row := db.QueryRow(query, args...)
	assert.NoError(t, row.Scan(&count))
	assert.Equal(t, 1, count)

	// Cleanup the que data
	delete := sqlbuilder.PostgreSQL.NewDeleteBuilder().DeleteFrom("que_jobs")
	delete.Where(delete.Equal("CAST (args ->> 'ID' AS INTEGER)", jobArgs.ID), delete.Equal("args ->> 'ACOID'", jobArgs.ACOID),
		delete.Equal("priority", priority))
	query, args = delete.Build()

	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}
