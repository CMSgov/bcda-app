package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

type mockTermination struct{}

func (mt mockTermination) Match(v any) bool {
	_, ok := v.(*models.Termination)
	return ok
}

var testACODenies = []string{"test001", "test002", "test005"}

func TestDenyACOsSuccess(t *testing.T) {
	ctx := context.Background()
	mock, err := pgxmock.NewConn()
	assert.Nil(t, err)
	defer mock.Close(ctx)

	mock.ExpectExec("^UPDATE acos SET termination_details = (.+)").
		WithArgs(mockTermination{}, testACODenies).
		WillReturnResult(pgxmock.NewResult("UPDATE", 3))

	err = denyACOs(ctx, mock, Payload{testACODenies})
	assert.Nil(t, err)
}

func TestDenyACOsQueryFailure(t *testing.T) {
	ctx := context.Background()
	mock, err := pgxmock.NewConn()
	assert.Nil(t, err)
	defer mock.Close(ctx)

	mock.ExpectExec("^UPDATE acos SET termination_details = (.+)").
		WithArgs(mockTermination{}, testACODenies).
		WillReturnError(errors.New("test error"))

	err = denyACOs(ctx, mock, Payload{testACODenies})
	assert.ErrorContains(t, err, "test error")
}

func TestDenyACOs_Integration(t *testing.T) {
	ctx := context.Background()

	dbURL, err := getDBURL()
	assert.Nil(t, err)

	conn, err := pgx.Connect(ctx, dbURL)
	assert.Nil(t, err)
	defer conn.Close(ctx)

	tx, err := conn.Begin(context.Background())
	assert.Nil(t, err)

	var ACO1, ACO2, ACO3, ACO4, ACO5 string
	err = tx.QueryRow(ctx, `INSERT INTO acos (cms_id, uuid, name) VALUES('test001', $1, 'ACO1') RETURNING id;`, uuid.New()).Scan(&ACO1)
	assert.Nil(t, err)
	err = tx.QueryRow(ctx, `INSERT INTO acos (cms_id, uuid, name) VALUES('test002', $1, 'ACO2') RETURNING id;`, uuid.New()).Scan(&ACO2)
	assert.Nil(t, err)
	err = tx.QueryRow(ctx, `INSERT INTO acos (cms_id, uuid, name) VALUES('test003', $1, 'ACO3') RETURNING id;`, uuid.New()).Scan(&ACO3)
	assert.Nil(t, err)
	err = tx.QueryRow(ctx, `INSERT INTO acos (cms_id, uuid, name) VALUES('test004', $1, 'ACO4') RETURNING id;`, uuid.New()).Scan(&ACO4)
	assert.Nil(t, err)
	err = tx.QueryRow(ctx, `INSERT INTO acos (cms_id, uuid, name) VALUES('test005', $1, 'ACO5') RETURNING id;`, uuid.New()).Scan(&ACO5)
	assert.Nil(t, err)

	err = denyACOs(ctx, tx, Payload{testACODenies})
	assert.Nil(t, err)

	rows, err := tx.Query(ctx, `SELECT id, cms_id, termination_details FROM acos WHERE id IN($1, $2, $3, $4, $5);`, ACO1, ACO2, ACO3, ACO4, ACO5)
	assert.Nil(t, err)
	defer rows.Close()

	for rows.Next() {
		var id string
		var cmsID string
		var td *models.Termination

		err = rows.Scan(&id, &cmsID, &td)
		assert.Nil(t, err)
		switch id {
		case ACO1, ACO2, ACO5:
			assert.Equal(t, td.DenylistType, models.Involuntary)
			assert.WithinDuration(t, td.CutoffDate, time.Now(), 1*time.Second)
			assert.WithinDuration(t, td.TerminationDate, time.Now(), 1*time.Second)
		case ACO3, ACO4:
			assert.Nil(t, td)
		default:
			t.Fail()
		}
	}

	// cleanup
	err = tx.Rollback(ctx)
	assert.Nil(t, err)
}
