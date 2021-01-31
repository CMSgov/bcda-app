package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/newrelic/go-agent/v3/newrelic"
	log "github.com/sirupsen/logrus"
)

// Timer provides methods for timing methods.
// Typical Usage scenario:
//		timer := metrics.GetTimer()
//		defer timer.Close()
//		ctx := metrics.NewContext(ctx, timer)
// 		ctx, close := metrics.NewParent(ctx)
// 		defer close()
// 		close1 := metrics.NewChild(ctx, "Ingest #1")
// 		// Perform Ingest #1 call
// 		close1()
// 		close2 := metrics.NewChild(ctx, "Ingest #2")
// 		// Perform Ingest #2 call
// 		close2()
type Timer interface {
	// new creates a new timer and embeds it into the returned context.
	// To start timing methods, caller should start with this call
	// and provide the returned context to newChild().
	new(parentCtx context.Context, name string) (ctx context.Context, close func())

	// newChild creates a timer (child) from the parent via the supplied context.
	newChild(parentCtx context.Context, name string) (close func())

	// Close cleans up all resources associated with the Timer. If any pending metrics
	// have not been reported, close will flush the result out.
	Close()
}

// To avoid collisions with other keys from other packages, we'll use a custom
// un-exported type for our context key.
type key int

const timerKey key = 0

// NewContext returns a new Context that carries the provided Timer
func NewContext(ctx context.Context, t Timer) context.Context {
	return context.WithValue(ctx, timerKey, t)
}

// NewParent creates a parent timer and embeds it into the returned context.
func NewParent(ctx context.Context, name string) (context.Context, func()) {
	t := fromContext(ctx)
	return t.new(ctx, name)
}

// NewChild creates a child timer from the parent found within the supplied context
func NewChild(ctx context.Context, name string) func() {
	t := fromContext(ctx)
	return t.newChild(ctx, name)
}

var defaultTimer = &noopTimer{}

// fromContext returns the Timer associated with the context.
// If no Timer is found on the context, a default no-op timer is returned.
func fromContext(ctx context.Context) Timer {
	t, ok := ctx.Value(timerKey).(Timer)
	if !ok {
		return defaultTimer
	}
	return t
}

func GetTimer() Timer {

	target := conf.GetEnv("DEPLOYMENT_TARGET")
	if target == "" {
		target = "local"
	}
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(fmt.Sprintf("BCDA-%s", target)),
		newrelic.ConfigLicense(conf.GetEnv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigEnabled(true),
		func(cfg *newrelic.Config) {
			cfg.HighSecurity = true
		},
	)

	if err != nil {
		log.Warnf("Failed to instantiate NeRelic application. Default to no-op timer. %s", err.Error())
		return &noopTimer{}
	}

	timeout := time.Duration(utils.GetEnvInt("NEW_RELIC_CONNECTION_TIMEOUT_SECONDS", 30)) * time.Second
	if err = app.WaitForConnection(timeout); err != nil {
		log.Warnf("Failed to establish connection to New Relic server in %s. Default to no-op timer.", timeout)
		return &noopTimer{}
	}

	log.Info("Using New Relic backed timer.")
	return &timer{app}
}

// validates that timer implements the interface
var _ Timer = &timer{}

type timer struct {
	nr *newrelic.Application
}

func (t *timer) new(parentCtx context.Context, name string) (ctx context.Context, close func()) {
	txn := t.nr.StartTransaction(name)
	ctx = newrelic.NewContext(parentCtx, txn)

	f := func() {
		txn.End()
	}
	return ctx, f
}

func (t *timer) newChild(parentCtx context.Context, name string) (close func()) {
	txn := newrelic.FromContext(parentCtx)
	if txn == nil {
		log.Warn("No transaction found. Cannot create child.")
		return noop
	}
	segment := txn.StartSegment(name)

	return func() {
		segment.End()
	}
}

func (t *timer) Close() {
	const SHUTDOWN_TIMEOUT = 30 * time.Second
	t.nr.Shutdown(SHUTDOWN_TIMEOUT)
}

// validates that noopTimer implements the interface
var _ Timer = &noopTimer{}

type noopTimer struct {
}

func (t *noopTimer) new(parentCtx context.Context, name string) (ctx context.Context, close func()) {
	return context.Background(), noop
}

func (t *noopTimer) newChild(parentCtx context.Context, name string) (close func()) {
	return noop
}

func (t *noopTimer) Close() {
}

func noop() {
}
