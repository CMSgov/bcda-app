/*
 Package main Beneficiary Claims Data API

 The Beneficiary Claims Data API (BCDA) allows downloading of claims data in accordance with the FHIR Bulk Data Export specification.

 If you have a Client ID and Secret you can use this page to explore the API.  To do this:
  1. Click the green "Authorize" button below and enter your Client ID and secret in the Basic Authentication boxes.
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
          name: Authorization
          description: The bulkData endpoints require a Bearer Token. 1) Put your credentials in Basic Authentication, 2) Request a bearer token from /auth/token, 3) Put "Bearer {TOKEN}" in this field (no quotes) using the bearer token retrieved in step 2
          in: header
     basic_auth:
          type: basic

 swagger:meta
*/
package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/bcdacli"
	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
)

func init() {
	isEtlMode := conf.GetEnv("BCDA_ETL_MODE")
	if isEtlMode != "true" {
		createAPIDirs()
	} else {
		createETLDirs()
	}

	if isEtlMode != "true" {
		log.API.Info("BCDA application is running in API mode.")
		monitoring.GetMonitor()
		log.API.Info(fmt.Sprintf(`Auth is made possible by %T`, auth.GetProvider()))
	} else {
		log.API.Info("BCDA application is running in ETL mode.")
	}

}

func createAPIDirs() {
	archive := conf.GetEnv("FHIR_ARCHIVE_DIR")
	err := os.MkdirAll(archive, 0744)
	if err != nil {
		log.API.Fatal(err)
	}
}

func createETLDirs() {
	pendingDeletionPath := conf.GetEnv("PENDING_DELETION_DIR")
	err := os.MkdirAll(pendingDeletionPath, 0744)
	if err != nil {
		log.API.Fatal(errors.Wrap(err, "Could not create CCLF file pending deletion directory"))
	}
}

func main() {
	app := bcdacli.GetApp()
	err := app.Run(os.Args)
	if err != nil {
		// Since the logs may be routed to a file,
		// ensure that the error makes it at least once to stdout
		fmt.Printf("Error occurred while executing command %s\n", err)
		log.API.Fatal(err)
	}
}
