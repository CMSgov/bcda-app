package monitoring

import (
	"fmt"
	"net/http"

    "github.com/CMSgov/bcda-app/conf"

	"github.com/newrelic/go-agent/v3/newrelic"
	log "github.com/sirupsen/logrus"
)

var a *apm

type apm struct {
	App *newrelic.Application
}

func (a apm) Start(msg string, w http.ResponseWriter, r *http.Request) *newrelic.Transaction {
	if a.App != nil {
		txn := a.App.StartTransaction(msg)
		txn.SetWebResponse(w)
		txn.SetWebRequestHTTP(r)
		return txn
	}
	return nil
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
			func(cfg *newrelic.Config) {
				cfg.HighSecurity = true
			},
		)
		if err != nil {
			log.Error(err)
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
