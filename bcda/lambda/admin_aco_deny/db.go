package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	log "github.com/sirupsen/logrus"

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

func denyACOs(ctx context.Context, conn PgxConnection, data payload) error {
	if len(data.DenyACOIDs) == 0 {
		return errors.New("no ACO IDs provided to deny")
	}

	cutoffDate := time.Now()
	if data.CutoffDate != nil && !data.CutoffDate.IsZero() {
		cutoffDate = *data.CutoffDate
	}

	termDate := cutoffDate
	if data.TerminationDate != nil && !data.TerminationDate.IsZero() {
		termDate = *data.TerminationDate
	}

	if termDate.After(cutoffDate) {
		return errors.New("termination_date cannot be after cutoff_date")
	}

	td := &models.Termination{
		TerminationDate: termDate,
		CutoffDate:      cutoffDate,
		DenylistType:    models.Involuntary,
	}

	query := "UPDATE acos SET termination_details = $1 WHERE cms_id = ANY ($2)"
	_, err := conn.Exec(ctx, query, td, data.DenyACOIDs)
	if err != nil {
		log.Errorf("Error running update query: %+v", err)
		return err
	}

	return nil
}
