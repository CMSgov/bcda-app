/*
 Package main Beneficiary Claims Data API

 The Beneficiary Claims Data API (BCDA) allows downloading of claims data in accordance with the FHIR Bulk Data Export specification.

 If you have a Client ID and Secret you can use this page to explore the API.  To do this:
  1. Click the green "Authorize" button below and enter your Client ID and secret in the Basic Authentication username and passsword boxes.
  2. Request a bearer token from /auth/token
  3. Click the green "Authorize" button below and put "Bearer {YOUR_TOKEN}" in the bearer_token box.

Until you click logout your token will be presented with every request made.  To make requests click on the
 "Try it out" button for the desired endpoint.


     Version: 1.0.0
     License: Public Domain https://github.com/CMSgov/bcda-app/blob/master/LICENSE.md
     Contact: bcapi@cms.hhs.gov

     Produces:
     - application/fhir+json
     - application/json

     SecurityDefinitions:
     bearer_token:
          type: apiKey
          name: The bulkData endpoints require a Bearer Token. 1) Put your credentials in Basic Authentication, 2) Request a bearer token from /auth/token, 3) Put "Bearer {TOKEN}" in this field (no quotes) using the bearer token retrieved in step 2
          in: header
     basic_auth:
          type: basic

 swagger:meta
*/
package main

import (
	"fmt"
	"github.com/pkg/errors"
	"os"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/bcdacli"
	"github.com/CMSgov/bcda-app/bcda/monitoring"

	log "github.com/sirupsen/logrus"
)

func init() {
	isEtlMode := os.Getenv("BCDA_ETL_MODE")
	if isEtlMode != "true" {
		createAPIDirs()
	} else {
		createETLDirs()
	}

	log.SetFormatter(&log.JSONFormatter{})
	filePath := os.Getenv("BCDA_ERROR_LOG")

	/* #nosec -- 0640 permissions required for Splunk ingestion */
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)

	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to open error log file; using default stderr")
	}

	if isEtlMode != "true" {
		log.Info("BCDA application is running in API mode.")
		monitoring.GetMonitor()
		log.Info(fmt.Sprintf(`Auth is made possible by %T`, auth.GetProvider()))
	} else {
		log.Info("BCDA application is running in ETL mode.")
	}

}

func createAPIDirs() {
	archive := os.Getenv("FHIR_ARCHIVE_DIR")
	err := os.MkdirAll(archive, 0744)
	if err != nil {
		log.Fatal(err)
	}
}

func createETLDirs() {
	pendingDeletionPath := os.Getenv("PENDING_DELETION_DIR")
	err := os.MkdirAll(pendingDeletionPath, 0744)
	if err != nil {
		log.Fatal(errors.Wrap(err,"Could not create CCLF file pending deletion directory"))
	}
}

func main() {
	app := bcdacli.GetApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
