package queueing

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/huandu/go-sqlbuilder"
	_ "github.com/jackc/pgx"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

func TestQueEnqueuer_Integration(t *testing.T) {
	// Need access to the queue database to ensure we've enqueued the job successfully
	db := database.QueueConnection

	priority := math.MaxInt16
	enqueuer := NewEnqueuer()
	jobArgs := models.JobEnqueueArgs{ID: int(rand.Int31()), ACOID: uuid.New()}
	alrJobArgs := models.JobAlrEnqueueArgs{
		ID:         1,
		CMSID:      "A1234",
		MBIs:       []string{"abd123abd01", "abd123abd02"},
		LowerBound: time.Now(),
		UpperBound: time.Now(),
	}
	assert.NoError(t, enqueuer.AddJob(jobArgs, priority))
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

func TestNewEnqueuer(t *testing.T) {
	origEnqueuer := conf.GetEnv("QUEUE_LIBRARY")

	// Test que-go implementation
	conf.SetEnv(t, "QUEUE_LIBRARY", "que-go")
	enq := NewEnqueuer()
	var expectedEnq queEnqueuer
	assert.IsType(t, expectedEnq, enq)

	// Test river implementation
	conf.SetEnv(t, "QUEUE_LIBRARY", "river")
	enq = NewEnqueuer()
	var expectedRiverEnq riverEnqueuer
	assert.IsType(t, expectedRiverEnq, enq)

	// If unmatched name: panic
	conf.SetEnv(t, "QUEUE_LIBRARY", "bad implementation name")
	assert.Panics(t, func() { enq = NewEnqueuer() }, "NO QUEUE LIBRARY SET")

	// If unset: panic
	conf.UnsetEnv(t, "QUEUE_LIBRARY")
	assert.Panics(t, func() { enq = NewEnqueuer() }, "NO QUEUE LIBRARY SET")

	// Reset env var
	conf.SetEnv(t, "QUEUE_LIBRARY", origEnqueuer)
}
