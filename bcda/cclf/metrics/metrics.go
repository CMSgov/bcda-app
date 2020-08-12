package metrics

import (
	"context"
	"fmt"
	"os"

	newrelic "github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"
)

// Timer provides methods for timing methods.
// Typical Usage scenario:
// 		ctx, close := Timer.New("Data Ingest")
// 		defer close()
// 		close1 := Timer.NewChild(ctx, "Ingest #1")
// 		// Perform Ingest #1 call
// 		close1()
// 		close2 := Timer.NewChild(ctx, "Ingest #2")
// 		// Perform Ingest #2 call
// 		close2()
type Timer interface {
	// New creates a new timer and embeds it into the returned context.
	// To start timing methods, caller should start with this call
	// and provide the returned context to NewChild().
	New(name string) (ctx context.Context, close func())

	// NewChild creates a timer (child) from the parent via the supplied context.
	NewChild(parentCtx context.Context, name string) (close func())
}

func GetTimer() Timer {
	target := os.Getenv("DEPLOYMENT_TARGET")
	if target == "" {
		target = "local"
	}
	config := newrelic.NewConfig(fmt.Sprintf("BCDA-%s", target), os.Getenv("NEW_RELIC_LICENSE_KEY"))
	config.Enabled = true
	config.HighSecurity = true
	app, err := newrelic.NewApplication(config)

	if err != nil {
		log.Warnf("Failed to instantiate NeRelic application. Default to no-op timer. %s", err.Error())
		return &noopTimer{}
	}

	return &timer{app}
}

// validates that timer implements the interface
var _ Timer = &timer{}

type timer struct {
	nr newrelic.Application
}

func (t *timer) New(name string) (ctx context.Context, close func()) {
	// Passing in nil http artifacts will allow us to time non-HTTP request
	txn := t.nr.StartTransaction(name, nil, nil)
	ctx = newrelic.NewContext(context.Background(), txn)

	f := func() {
		if err := txn.End(); err != nil {
			log.Warnf("Error occurred when ending transaction %s", err.Error())
		}
	}
	return ctx, f
}

func (t *timer) NewChild(parentCtx context.Context, name string) (close func()) {
	txn := newrelic.FromContext(parentCtx)
	if txn == nil {
		log.Warn("No transaction found. Cannot create child.")
		return noop
	}
	segment := newrelic.StartSegment(txn, name)

	return func() {
		if err := segment.End(); err != nil {
			log.Warnf("Error occurred when ending segment %s", err.Error())
		}
	}
}

// validates that noopTimer implements the interface
var _ Timer = &noopTimer{}

type noopTimer struct {
}

func (t *noopTimer) New(name string) (ctx context.Context, close func()) {
	return context.Background(), noop
}

func (t *noopTimer) NewChild(parentCtx context.Context, name string) (close func()) {
	return noop
}

func noop() {
}
