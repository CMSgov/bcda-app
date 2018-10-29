package monitoring

import (
	"os"
	"strconv"

	newrelic "github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"
)

type apm struct {
	App newrelic.Application
}

var a *apm

func GetMonitor(enabled string) *apm {
	if a == nil {
		config := newrelic.NewConfig("BCDA-dev", os.Getenv("NEW_RELIC_LICENSE_KEY"))
		e, err := strconv.ParseBool(enabled)
		if err != nil {
			log.Error(err)
		}
		config.Enabled = e
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

func End(txn newrelic.Transaction) {
	err := txn.End()
	if err != nil {
		log.Error(err)
	}
}
