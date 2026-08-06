package service

import (
	"context"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/models/fhir/r4"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcdaworker/worker"
	"github.com/CMSgov/bcda-app/conf"
)

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

// SetupWarningsAndInfoFile finds or creates an info file to house generic warnings, issues, etc for a given job.
// It will also find or create a jobkey.  There should only be 1 warnings and info file per job.
func SetupWarningsAndInfoFile(ctx context.Context, pgxRepo *postgres.PgxRepository, jobID uint) error {
	payloadPath := fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_PAYLOAD_DIR"), jobID)

	err := worker.CreateDir(payloadPath)
	if err != nil {
		return fmt.Errorf("error creating payload directory: %w", err)
	}

	err = pgxRepo.FindOrCreateWarningAndInfoJobKey(ctx, jobID)
	if err != nil {
		return fmt.Errorf("error creating warnings and info job key: %w", err)
	}

	return nil
}
