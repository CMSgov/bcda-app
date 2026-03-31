package queueing

import (
	"context"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/stretchr/testify/assert"
)

type TestJobArgs struct {
}

func (args TestJobArgs) Kind() string {
	return "TestJob"
}

type TestJobWorker struct {
	river.WorkerDefaults[TestJobArgs]
}

func (w *TestJobWorker) Timeout(*river.Job[TestJobArgs]) time.Duration {
	return 5 * time.Second
}

func (w *TestJobWorker) Work(ctx context.Context, rjob *river.Job[TestJobArgs]) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			time.Sleep(1 * time.Second)
			return nil
		}
	}
}

func TestWorkerRespectsParentContext(t *testing.T) {
	bgCtx := context.Background()

	// test Timeout
	ctx, cancel := context.WithTimeout(bgCtx, (1 * time.Nanosecond))
	t.Cleanup(cancel)
	err := (&TestJobWorker{}).Work(ctx, &river.Job[TestJobArgs]{Args: TestJobArgs{}})
	assert.Equal(t, ctx.Err(), err)

	// test Deadline
	ctx, cancel = context.WithDeadline(bgCtx, time.Now().Add(1*time.Nanosecond))
	t.Cleanup(cancel)
	err = (&TestJobWorker{}).Work(ctx, &river.Job[TestJobArgs]{Args: TestJobArgs{}})
	assert.Equal(t, ctx.Err(), err)

	// test No Cancellation/Timeout/Deadline
	ctx, cancel = context.WithTimeout(bgCtx, (3 * time.Second))
	t.Cleanup(cancel)
	err = (&TestJobWorker{}).Work(ctx, &river.Job[TestJobArgs]{Args: TestJobArgs{}})
	assert.Nil(t, err)
}
