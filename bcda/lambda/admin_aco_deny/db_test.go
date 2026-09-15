package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
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

type mockTerminationWithCutoff struct {
	expectedCutoff time.Time
}

func (m mockTerminationWithCutoff) Match(v any) bool {
	td, ok := v.(*models.Termination)
	if !ok {
		return false
	}
	return td.CutoffDate.Equal(m.expectedCutoff) && td.TerminationDate.Equal(m.expectedCutoff)
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

	err = denyACOs(ctx, mock, payload{DenyACOIDs: testACODenies})
	assert.Nil(t, err)
}

func TestDenyACOsWithCustomCutoffDate(t *testing.T) {
	ctx := context.Background()
	mock, err := pgxmock.NewConn()
	assert.Nil(t, err)
	defer mock.Close(ctx)

	futureCutoff := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second)
	mock.ExpectExec("^UPDATE acos SET termination_details = (.+)").
		WithArgs(mockTerminationWithCutoff{expectedCutoff: futureCutoff}, testACODenies).
		WillReturnResult(pgxmock.NewResult("UPDATE", 3))

	err = denyACOs(ctx, mock, payload{DenyACOIDs: testACODenies, CutoffDate: &futureCutoff})
	assert.Nil(t, err)
}

type mockTerminationWithDates struct {
	expectedCutoff      time.Time
	expectedTermination time.Time
}

func (m mockTerminationWithDates) Match(v any) bool {
	td, ok := v.(*models.Termination)
	if !ok {
		return false
	}
	return td.CutoffDate.Equal(m.expectedCutoff) && td.TerminationDate.Equal(m.expectedTermination)
}

func TestDenyACOsWithCustomTerminationDate(t *testing.T) {
	ctx := context.Background()
	mock, err := pgxmock.NewConn()
	assert.Nil(t, err)
	defer mock.Close(ctx)

	termDate := time.Now().Add(10 * 24 * time.Hour).Truncate(time.Second)
	cutoffDate := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second)
	mock.ExpectExec("^UPDATE acos SET termination_details = (.+)").
		WithArgs(mockTerminationWithDates{expectedCutoff: cutoffDate, expectedTermination: termDate}, testACODenies).
		WillReturnResult(pgxmock.NewResult("UPDATE", 3))

	err = denyACOs(ctx, mock, payload{
		DenyACOIDs:      testACODenies,
		CutoffDate:      &cutoffDate,
		TerminationDate: &termDate,
	})
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

	err = denyACOs(ctx, mock, payload{DenyACOIDs: testACODenies})
	assert.ErrorContains(t, err, "test error")
}

func TestDenyACOsNoACOIDs(t *testing.T) {
	ctx := context.Background()
	mock, err := pgxmock.NewConn()
	assert.Nil(t, err)
	defer mock.Close(ctx)

	err = denyACOs(ctx, mock, payload{DenyACOIDs: []string{}})
	assert.EqualError(t, err, "no ACO IDs provided to deny")
}

func TestDenyACOsTerminationDateAfterCutoffDate(t *testing.T) {
	ctx := context.Background()
	mock, err := pgxmock.NewConn()
	assert.Nil(t, err)
	defer mock.Close(ctx)

	cutoffDate := time.Now()
	termDate := cutoffDate.Add(24 * time.Hour)

	err = denyACOs(ctx, mock, payload{
		DenyACOIDs:      testACODenies,
		CutoffDate:      &cutoffDate,
		TerminationDate: &termDate,
	})
	assert.EqualError(t, err, "termination_date cannot be after cutoff_date")
}

func TestDenyACOs_Integration(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	assert.Nil(t, err)
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
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

	err = denyACOs(ctx, tx, payload{DenyACOIDs: testACODenies})
	assert.Nil(t, err)

	rows, err := tx.Query(ctx, `SELECT id, cms_id, termination_details FROM acos WHERE id IN($1, $2, $3, $4, $5);`, ACO1, ACO2, ACO3, ACO4, ACO5)
	assert.Nil(t, err)
	defer rows.Close()

	i := 0
	for rows.Next() {
		var id string
		var cmsID string
		var tdJSON []byte
		i++

		err = rows.Scan(&id, &cmsID, &tdJSON)
		assert.Nil(t, err)
		switch id {
		case ACO1, ACO2, ACO5:
			assert.NotNil(t, tdJSON, "termination_details should not be null for denied ACOs")
			var td models.Termination
			err = json.Unmarshal(tdJSON, &td)
			assert.Nil(t, err)
			assert.Equal(t, td.DenylistType, models.Involuntary)
			assert.WithinDuration(t, td.CutoffDate, time.Now(), 1*time.Second)
			assert.WithinDuration(t, td.TerminationDate, time.Now(), 1*time.Second)
		case ACO3, ACO4:
			assert.Nil(t, tdJSON, "termination_details should be null for non-denied ACOs")
		default:
			t.Fail()
		}
	}

	// double check we are finding the appropriate amount of rows
	assert.Equal(t, i, 5)

	// cleanup
	err = tx.Rollback(ctx)
	assert.Nil(t, err)
}
