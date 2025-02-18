package main

import (
	"context"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	log "github.com/sirupsen/logrus"
)

// this implements both pgx transactions (tx) as well as pgx connections
type PgxConnection interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Prepare(context.Context, string, string) (*pgconn.StatementDescription, error)
}

func createACO(ctx context.Context, conn PgxConnection, aco models.ACO) error {
	query := `INSERT INTO acos (uuid, cms_id, client_id, name, termination_details) VALUES($1, $2, $3, $4, $5) RETURNING id;`
	_, err := conn.Exec(ctx, query, aco.UUID, aco.CMSID, aco.ClientID, aco.Name, &models.Termination{})
	if err != nil {
		log.Errorf("Error running update query: %+v", err)
		return err
	}

	return nil
}
