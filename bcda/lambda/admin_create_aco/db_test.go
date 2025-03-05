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
	"github.com/stretchr/testify/suite"
)

type CreateACOTestSuite struct {
	suite.Suite
	tx   pgx.Tx
	conn pgx.Conn
	ctx  context.Context
}

func (c *CreateACOTestSuite) SetupTest() {
	c.ctx = context.Background()

	params, err := getAWSParams()
	if err != nil {
		assert.FailNow(c.T(), "Failed to get AWS Params")
	}

	conn, err := pgx.Connect(c.ctx, params.dbURL)
	if err != nil {
		assert.FailNow(c.T(), "Failed to setup pgx connection")
	}

	c.tx, err = conn.Begin(c.ctx)
	if err != nil {
		assert.FailNow(c.T(), "Failed to begin transaction")
	}
}

func (c *CreateACOTestSuite) TeardownTest() {
	// cleanup
	err := c.tx.Rollback(c.ctx)
	if err != nil {
		assert.FailNow(c.T(), "Failed to rollback transaction")
	}

	c.conn.Close(context.Background())
}

func TestCreateACOTestSuite(t *testing.T) {
	suite.Run(t, new(CreateACOTestSuite))
}

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

func (c *CreateACOTestSuite) TestCreateACO_Integration() {
	var (
		name  = "Test ACO 1"
		cmsId = "TEST501"
	)
	testACO := createTestACOModel(name, cmsId)

	c.T().Run("Valid ACO model", func(t *testing.T) {
		err := createACO(c.ctx, c.tx, testACO)
		assert.Nil(t, err)
	})

	// check that the ACO is within the DB
	var count int
	err := c.tx.QueryRow(c.ctx, "SELECT COUNT(*) FROM acos WHERE (name, cms_id) = ($1, $2)", name, cmsId).Scan(&count)
	if err != nil {
		assert.FailNow(c.T(), "Failed to get count of ACO")
	}
	assert.Equal(c.T(), count, 1)
}

func (c *CreateACOTestSuite) TestCreateACO_DupCMSID_Integration() {
	// put original ACO into DB
	_, err := c.tx.Exec(c.ctx, "INSERT INTO acos (name, uuid, cms_id) VALUES ('TEST ACO 2', $1, 'TEST501') RETURNING id;", uuid.New())
	if err != nil {
		assert.FailNow(c.T(), "Failed to insert ACO into database")
	}

	// insert ACO with duplicate CMS ID
	c.T().Run("Duplicate CMSID", func(t *testing.T) {
		err = createACO(c.ctx, c.tx, createTestACOModel("Test ACO 3", "TEST501"))
		assert.Contains(t, err.Error(), "duplicate key")
	})
}

func createTestACOModel(name string, cmsId string) models.ACO {
	testUuid := uuid.NewRandom()
	testClientId := testUuid.String()
	td := models.Termination{}
	cmsIdPtr := &cmsId
	aco := models.ACO{Name: name, CMSID: cmsIdPtr, UUID: testUuid, ClientID: testClientId, TerminationDetails: &td}
	return aco
}
