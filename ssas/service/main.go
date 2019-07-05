package main

import (
	"github.com/CMSgov/bcda-app/ssas"
)

func main() {
	ssas.Logger.Info("Future home of the System-to-System Authentication Service")
	ssas.Logger.Infof("SSAS gave me %s", ssas.Provide())
}

func hello() string {
	return "hello SSAS"
}

