package service

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/r4"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/conf"
)

var HasSystemTypeRegex = regexp.MustCompile(`.*\?_typeFilter=.*\?_tag=.*\/System-Type.*`)
var WarningDefaultSystemType = r4.OperationOutcome{
	ResourceType: "OperationOutcome",
	Issue: []r4.Issue{
		{
			Code: "processing",
			Details: &r4.CodeableConcept{
				Text: "Default System-Type behavior includes only claims from NCH and DDPS in this export.",
			},
			Severity: "warning",
		},
	},
}

// []byte`{"resourceType":"OperationOutcome","issue":[{"severity":"warning","code":"processing","details":{"text":"Default System-Type behavior includes only claims from NCH and DDPS in this export."}}]}`

// SetupWarningsAndInfoFile finds or creates the warnings and info file to house generic warnings and issues for a given job.
// It will also find or create a jobkey.  There should only be 1 warnings and info file per job.
func SetupWarningsAndInfoFile(ctx context.Context, pgxRepo *postgres.PgxRepository, jobID uint) (string, error) {
	payloadPath := fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_PAYLOAD_DIR"), jobID)
	filePath := payloadPath + "/" + constants.WarningsAndInfoFileName

	err := worker.CreateDir(payloadPath)
	if err != nil {
		return "", fmt.Errorf("error creating payload directory: %w", err)
	}

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return "", fmt.Errorf("error opening or creating warnings and info file: %w", err)
	}
	defer file.Close()

	err = pgxRepo.FindOrCreateWarningAndInfoJobKey(ctx, jobID)
	if err != nil {
		return "", fmt.Errorf("error creating warnings and info job key: %w", err)
	}

	return filePath, nil
}
