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

var insertACOQuery = `INSERT INTO acos (uuid, cms_id, client_id, name, termination_details) VALUES($1, $2, $3, $4, $5) RETURNING id;`

func createACO(ctx context.Context, conn PgxConnection, aco models.ACO) error {
	_, err := conn.Exec(ctx, insertACOQuery, aco.UUID, aco.CMSID, aco.ClientID, aco.Name, &models.Termination{})
	if err != nil {
		log.Errorf("Error running insert ACO query: %+v", err)
		return err
	}

	return nil
}
