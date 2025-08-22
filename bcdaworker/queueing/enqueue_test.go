package queueing

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/huandu/go-sqlbuilder"
	"github.com/pborman/uuid"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertest"
	"github.com/stretchr/testify/assert"
)

func TestEnqueuerImplementation(t *testing.T) {
	// Test river implementation
	enq := NewEnqueuer(nil, nil)
	var expectedRiverEnq riverEnqueuer
	assert.IsType(t, expectedRiverEnq, enq)
}

func TestRiverEnqueuer_Integration(t *testing.T) {
	defer func(origEnqueuer string) {
		conf.SetEnv(t, "QUEUE_LIBRARY", origEnqueuer)
	}(conf.GetEnv("QUEUE_LIBRARY"))

	conf.SetEnv(t, "QUEUE_LIBRARY", "river")

	// Need access to the queue database to ensure we've enqueued the job successfully
	db := database.Connect()
	pool := database.ConnectPool()

	enqueuer := NewEnqueuer(db, pool)
	jobID, e := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	if e != nil {
		t.Fatalf("failed to generate job ID: %v\n", e)
	}
	jobArgs := worker_types.JobEnqueueArgs{ID: int(jobID.Int64()), ACOID: uuid.New()}

	ctx := context.Background()
	assert.NoError(t, enqueuer.AddJob(ctx, jobArgs, 3))

	// Use river test helper to assert job was inserted
	checkJob := rivertest.RequireInserted(ctx, t, riverpgxv5.New(pool), jobArgs, nil)
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
