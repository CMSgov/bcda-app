package monitoring

import (
	"fmt"
	"net/http"
	"context"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	
	"github.com/newrelic/go-agent/v3/newrelic"
)

var a *apm

type apm struct {
	App *newrelic.Application
}

// QUESTION: CAN I NOT MAKE THE LAST TWO PARAMS OPTIONAL?
// QUESTION: HOW IS TRANSACTION ADDED TO CONTEXT? IS IT?
func (a apm) Start(msg string, w http.ResponseWriter, r *http.Request) *newrelic.Transaction {
	if a.App != nil {
		txn := a.App.StartTransaction(msg)
		txn.SetWebResponse(w)
		txn.SetWebRequestHTTP(r)
		return txn
	}
	return nil
}

// ALTERNATIVE TO START BC OF QUESTION ON LINE 20
func (a apm) NewTransaction(name string) (*newrelic.Transaction) {
	if a.App != nil {
		txn := a.App.StartTransaction(name) // transaction trace
		// ctx := newrelic.NewContext(context.Background(), txn) // parent context
		return txn
	}
	return nil
}

func NewSpan(parentCtx context.Context, name string) (close func()) {
	txn := newrelic.FromContext(parentCtx)
	if txn == nil {
		log.API.Warn("No transaction found. Cannot create child.")
	}
	segment := txn.StartSegment(name)

	return func() {
		segment.End()
	}
}

func (a apm) End(txn *newrelic.Transaction) {
	if a.App != nil {
		txn.End()
	}
}

func GetMonitor() *apm {
	if a == nil {
		target := conf.GetEnv("DEPLOYMENT_TARGET")
		if target == "" {
			target = "local"
		}
		app, err := newrelic.NewApplication(
			newrelic.ConfigAppName(fmt.Sprintf("BCDA-%s", target)),
			newrelic.ConfigLicense(conf.GetEnv("NEW_RELIC_LICENSE_KEY")),
			newrelic.ConfigEnabled(true),
			newrelic.ConfigDistributedTracerEnabled(true),
			func(cfg *newrelic.Config) {
				cfg.HighSecurity = true
			},
		)
		if err != nil {
			log.API.Error(err)
		}
		a = &apm{
			App: app,
		}
	}
	return a
}

func (a apm) WrapHandler(pattern string, h http.HandlerFunc) (string, func(http.ResponseWriter, *http.Request)) {
	if a.App != nil {
		return newrelic.WrapHandleFunc(a.App, pattern, h)
	}
	return pattern, h
}