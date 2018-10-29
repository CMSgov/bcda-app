package monitoring

import (
	"fmt"
	"net/http"
	"os"

	newrelic "github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"
)

var a *apm

type apm struct {
	App newrelic.Application
}

func (a apm) Start(msg string, w http.ResponseWriter, r *http.Request) newrelic.Transaction {
	if a.App != nil {
		return a.App.StartTransaction(msg, w, r)
	}
	return nil
}

func (a apm) End(txn newrelic.Transaction) {
	if a.App != nil {
		err := txn.End()
		if err != nil {
			log.Error(err)
		}
	}
}

func GetMonitor() *apm {
	if a == nil {
		target := os.Getenv("DEPLOYMENT_TARGET")
		if target == "" {
			target = "local"
		}
		config := newrelic.NewConfig(fmt.Sprintf("BCDA-%s", target), os.Getenv("NEW_RELIC_LICENSE_KEY"))
		config.Enabled = true
		config.HighSecurity = true
		app, err := newrelic.NewApplication(config)
		if err != nil {
			log.Error(err)
		}
		a = &apm{
			App: app,
		}
	}
	return a
}
