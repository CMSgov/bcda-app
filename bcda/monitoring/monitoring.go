package monitoring

import (
	"fmt"
	"net/http"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var a *apm

type apm struct {
	App *newrelic.Application
}

// func (a apm) Start(msg string, w http.ResponseWriter, r *http.Request) *newrelic.Transaction {
// 	if a.App != nil {
// 		txn := a.App.StartTransaction(msg)
// 		//s := newrelic.StartExternalSegment(txn, r)
// 		txn.SetWebResponse(w)
// 		txn.SetWebRequestHTTP(r)
// 		return txn
// 	}
// 	return nil
// }

// func (a apm) End(txn *newrelic.Transaction) {
// 	if a.App != nil {
// 		txn.End()
// 	}
// }

func (a apm) Start(msg string, r *http.Request) *newrelic.ExternalSegment {
	if a.App != nil {
		txn := newrelic.FromContext(r.Context())
		s := newrelic.StartExternalSegment(txn, r)
		return s
	}
	return nil
}

func (a apm) End(s *newrelic.ExternalSegment) {
	if a.App != nil {
		s.End()
	}
}

func GetMonitor() *apm {
	if a == nil {
		target := conf.GetEnv("DEPLOYMENT_TARGET")
		enabled := true
		if target == "" {
			target = "local"
			enabled = false
		}
		app, err := newrelic.NewApplication(
			newrelic.ConfigEnabled(enabled),
			newrelic.ConfigAppName(fmt.Sprintf("BCDA-%s", target)),
			newrelic.ConfigLicense(conf.GetEnv("NEW_RELIC_LICENSE_KEY")),
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
