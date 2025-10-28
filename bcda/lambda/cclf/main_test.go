package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AttributionImportMainSuite struct {
	suite.Suite
	db *sql.DB
}

func (s *AttributionImportMainSuite) SetupSuite() {
	s.db = database.Connect()
}

func (s *AttributionImportMainSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
}

func TestAttributionImportMainSuite(t *testing.T) {
	suite.Run(t, new(AttributionImportMainSuite))
}

func (s *AttributionImportMainSuite) TestImportCCLFDirectory() {
	targetACO := "A0001"
	assert := assert.New(s.T())
	cfg := testUtils.TestAWSConfig(s.T())
	s3Client := testUtils.TestS3Client(s.T(), cfg)
	ssmClient := testUtils.TestSSMClient(s.T(), cfg)
	pool := database.ConnectPool()

	env := uuid.NewUUID()
	cleanupEnv := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{
		{Name: "CCLF_REF_DATE", Value: "181125"},
		{Name: "ENV", Value: env.String()},
	})
	defer cleanupEnv()

	cleanupParam1 := testUtils.SetParameter(s.T(), fmt.Sprintf("/cclf-import/bcda/%s/bfd-bucket-role-arn", env), "arn:aws:iam::000000000000:user/fake-arn")
	cleanupParam2 := testUtils.SetParameter(s.T(), fmt.Sprintf("/bcda/%s/api/DATABASE_URL", env), "postgresql://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable")
	defer cleanupParam1()
	defer cleanupParam2()

	type test struct {
		path         string
		filename     string
		err          error
		expectedLogs []string
	}

	tests := []test{
		{path: "../../../shared_files/cclf/archives/valid/", filename: "cclf/archives/valid/T.BCD.A0001.ZCY18.D181120.T1000000", expectedLogs: []string{"Successfully imported", "Failed to import 0 files.", "Skipped 0 files."}},
		{path: "../../../shared_files/cclf/archives/invalid_bcd/", filename: "cclf/archives/invalid_bcd/P.BCD.A0009.ZCY18.D181120.T0001000", err: errors.New("files skipped or failed import. See logs for more details"), expectedLogs: []string{}},
		{path: "../../../shared_files/cclf/archives/skip/", filename: "cclf/archives/skip/T.BCD.ACOB.ZC0Y18.D181120.T0001000", expectedLogs: []string{"Successfully imported 0 files.", "Failed to import 0 files.", "Skipped 0 files."}},
	}

	for _, tc := range tests {
		fmt.Printf("\n----- path: %s\n", tc.path)
		postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, targetACO)
		defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, targetACO)

		path, cleanup := testUtils.CopyToS3(s.T(), tc.path)
		defer cleanup()

		res, err := handleCclfImport(context.Background(), pool, s3Client, ssmClient, path)

		if tc.err == nil {
			assert.Nil(err)
		} else {
			assert.NotNil(err)
			assert.Contains(err.Error(), tc.err.Error())
		}

		for _, entry := range tc.expectedLogs {
			assert.Contains(res, entry)
		}
	}
}

func (s *AttributionImportMainSuite) TestHandlerMissingS3AssumeRoleArn() {
	assert := assert.New(s.T())
	cfg := testUtils.TestAWSConfig(s.T())
	s3Client := testUtils.TestS3Client(s.T(), cfg)
	ssmClient := testUtils.TestSSMClient(s.T(), cfg)
	pool := database.ConnectPool()

	_, err := handleCclfImport(context.Background(), pool, s3Client, ssmClient, "asdf")
	assert.Contains(err.Error(), "error retrieving parameter /cclf-import/bcda/local/bfd-bucket-role-arn from parameter store")
}
