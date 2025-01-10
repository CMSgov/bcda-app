package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	log "github.com/sirupsen/logrus"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/models"
)

// this implements both pgx transactions (tx) as well as pgx connections
type PgxConnection interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Prepare(context.Context, string, string) (*pgconn.StatementDescription, error)
}

func denyACOs(ctx context.Context, conn PgxConnection, payload Payload) error {
	td := &models.Termination{
		TerminationDate: time.Now(),
		CutoffDate:      time.Now(),
		DenylistType:    models.Involuntary,
	}

	query := "UPDATE acos SET termination_details = $1 WHERE cms_id = ANY ($2)"
	_, err := conn.Exec(ctx, query, td, payload.DenyACOIDs)
	if err != nil {
		log.Errorf("Error running update query: %+v", err)
		return err
	}

	return nil
}

func getDBURL() (string, error) {
	env := conf.GetEnv("ENV")

	if env == "local" {
		return conf.GetEnv("DATABASE_URL"), nil
	}

	bcdaSession, err := bcdaaws.NewSession("", os.Getenv("LOCAL_STACK_ENDPOINT"))
	if err != nil {
		return "", err
	}

	param, err := bcdaaws.GetParameter(bcdaSession, fmt.Sprintf("/bcda/%s/api/DATABASE_URL", env))
	if err != nil {
		return "", err
	}

	return param, nil
}
