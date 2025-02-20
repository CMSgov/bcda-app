package main

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/jackc/pgx/v5"
	"github.com/pborman/uuid"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

var (
	cms_id       = "TEST002"
	testACO      = &models.ACO{UUID: uuid.NewRandom(), CMSID: &cms_id, Name: "TESTACO", ClientID: "", TerminationDetails: &models.Termination{}}
	literalRegex = regexp.QuoteMeta(insertACOQuery)
)

func TestCreateACOSuccess(t *testing.T) {
	ctx := context.Background()
	mockConn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	defer mockConn.Close(ctx)

	mockConn.ExpectExec(literalRegex).
		WithArgs(testACO.UUID, testACO.CMSID, testACO.ClientID, testACO.Name, testACO.TerminationDetails).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = createACO(ctx, mockConn, *testACO)
	assert.Nil(t, err)
}

func TestCreateACOQueryFailure(t *testing.T) {
	ctx := context.Background()
	mockConn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	defer mockConn.Close(ctx)

	mockConn.ExpectExec(literalRegex).
		WithArgs(testACO.UUID, testACO.CMSID, testACO.ClientID, testACO.Name, testACO.TerminationDetails).
		WillReturnError(errors.New("test error"))

	err = createACO(ctx, mockConn, *testACO)
	assert.ErrorContains(t, err, "test error")
}

func TestCreateACO_Integration(t *testing.T) {
	ctx := context.Background()

	params, err := getAWSParams()
	assert.Nil(t, err)

	conn, err := pgx.Connect(ctx, params.dbURL)
	assert.Nil(t, err)
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	assert.Nil(t, err)

	tests := []struct {
		name string
		aco  models.ACO
		err  string
	}{
		{"Valid ACO model", createACOModel("Test 501", "TEST501"), ""},
		{"Duplicate ACO Name", createACOModel("Test 501", "TEST502"), "Duplicate ACO Model"},
		{"Duplicate CMSID", createACOModel("Test 503", "TEST502"), ""},
		{"Duplicate ACO Name and CMSID", createACOModel("Test 503", "TEST502"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = createACO(ctx, tx, tt.aco)
			if tt.err != "" {
				assert.Contains(t, err.Error(), tt.err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

	// cleanup
	tx.Rollback(ctx)
	assert.Nil(t, err)
}

func createACOModel(name string, cmsId string) models.ACO {
	testUuid := uuid.NewRandom()
	testClientId := testUuid.String()
	td := models.Termination{}
	return models.ACO{Name: name, CMSID: &cmsId, UUID: testUuid, ClientID: testClientId, TerminationDetails: &td}
}

// func TestCreateACO_Integration(t *testing.T) {
// 	ctx := context.Background()
//
// 	params, err := getAWSParams()
// 	assert.Nil(t, err)
//
// 	conn, err := pgx.Connect(ctx, params.dbURL)
// 	assert.Nil(t, err)
// 	defer conn.Close(ctx)
//
// 	tx, err := conn.Begin(ctx)
// 	assert.Nil(t, err)
//
// 	// aco.UUID, aco.CMSID, aco.ClientID, aco.Name, &models.Termination{}
// 	id := uuid.NewRandom()
// 	clientid := id.String()
// 	name := "TEST501"
// 	err = tx.QueryRow(ctx, `INSERT INTO acos (uuid, cms_id, client_id, name, termination_details) VALUES($1, $2, $3, $4, $5) RETURNING id;`, id, clientid, name)
// 	assert.Nil(t, err)
//
// 	// cleanup
// 	err = tx.Rollback(ctx)
// 	assert.Nil(t, err)
// }
