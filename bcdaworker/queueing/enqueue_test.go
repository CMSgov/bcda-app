package queueing

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/huandu/go-sqlbuilder"
	_ "github.com/jackc/pgx"
	"github.com/pborman/uuid"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertest"
	"github.com/stretchr/testify/assert"
)

func TestEnqueuerImplementation(t *testing.T) {
	defer func(origEnqueuer string) {
		conf.SetEnv(t, "QUEUE_LIBRARY", origEnqueuer)
	}(conf.GetEnv("QUEUE_LIBRARY"))

	// Test que-go implementation (default)
	enq := NewEnqueuer()
	var expectedEnq queEnqueuer
	assert.IsType(t, expectedEnq, enq)

	// Test river implementation
	conf.SetEnv(t, "QUEUE_LIBRARY", "river")
	enq = NewEnqueuer()
	var expectedRiverEnq riverEnqueuer
	assert.IsType(t, expectedRiverEnq, enq)

	// If unset use default
	conf.UnsetEnv(t, "QUEUE_LIBRARY")
	enq = NewEnqueuer()
	assert.IsType(t, expectedEnq, enq)
}

func TestQueEnqueuer_Integration(t *testing.T) {
	// Need access to the queue database to ensure we've enqueued the job successfully
	db := database.QueueConnection

	priority := math.MaxInt16
	enqueuer := NewEnqueuer()
	jobID, e := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	if e != nil {
		t.Fatalf("failed to generate job ID: %v", e)
	}
	jobArgs := models.JobEnqueueArgs{ID: int(jobID.Int64()), ACOID: uuid.New()}
	alrJobArgs := models.JobAlrEnqueueArgs{
		ID:         1,
		CMSID:      "A1234",
		MBIs:       []string{"abd123abd01", "abd123abd02"},
		LowerBound: time.Now(),
		UpperBound: time.Now(),
	}
	fmt.Printf("Que Test Job args: %+v", jobArgs)
	ctx := context.Background()
	assert.NoError(t, enqueuer.AddJob(ctx, jobArgs, priority))
	assert.NoError(t, enqueuer.AddAlrJob(alrJobArgs, priority))

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

func TestRiverEnqueuer_Integration(t *testing.T) {
	defer func(origEnqueuer string) {
		conf.SetEnv(t, "QUEUE_LIBRARY", origEnqueuer)
	}(conf.GetEnv("QUEUE_LIBRARY"))

	conf.SetEnv(t, "QUEUE_LIBRARY", "river")

	// Need access to the queue database to ensure we've enqueued the job successfully
	db := database.Connection

	enqueuer := NewEnqueuer()
	jobID, e := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	if e != nil {
		t.Fatalf("failed to generate job ID: %v\n", e)
	}
	jobArgs := models.JobEnqueueArgs{ID: int(jobID.Int64()), ACOID: uuid.New()}

	ctx := context.Background()
	assert.NoError(t, enqueuer.AddJob(ctx, jobArgs, 3))

	// Use river test helper to assert job was inserted
	checkJob := rivertest.RequireInserted(ctx, t, riverpgxv5.New(database.Pgxv5Pool), jobArgs, nil)
	assert.NotNil(t, checkJob)

	// Also Verify that we've inserted the river job as expected via DB queries
	sb := sqlbuilder.PostgreSQL.NewSelectBuilder().Select("COUNT(1)").From("river_job")
	sb.Where(
		sb.Equal("CAST (args ->> 'ID' AS INTEGER)", jobArgs.ID),
		sb.Equal("args ->> 'ACOID'", jobArgs.ACOID),
		sb.Equal("priority", 3),
	)

	var count int
	query, args := sb.Build()
	row := db.QueryRow(query, args...)
	assert.NoError(t, row.Scan(&count))
	assert.Equal(t, 1, count)

	// Cleanup the queue data
	delete := sqlbuilder.PostgreSQL.NewDeleteBuilder().DeleteFrom("river_job")
	delete.Where(
		delete.Equal("CAST (args ->> 'ID' AS INTEGER)", jobArgs.ID),
		delete.Equal("args ->> 'ACOID'", jobArgs.ACOID),
		delete.Equal("priority", 3),
	)
	query, args = delete.Build()

	_, err := db.Exec(query, args...)
	assert.NoError(t, err)
}
