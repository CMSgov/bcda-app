package main

import (
	"context"
	"errors"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type CclfImportMainSuite struct {
	suite.Suite
}

func TestCclfImportMainSuite(t *testing.T) {
	suite.Run(t, new(CclfImportMainSuite))
}

func (s *CclfImportMainSuite) TestImportCCLFDirectory() {
	targetACO := "A0002"
	assert := assert.New(s.T())

	cleanup := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{{Name: "CCLF_REF_DATE", Value: "181125"}})
	defer cleanup()

	cleanupParams := testUtils.SetParameters(s.T(), []testUtils.AwsParameter{
		{Name: "/cclf-import/bcda/local/bfd-bucket-role-arn", Value: "arn:aws:iam::000000000000:user/fake-arn", Type: "String"},
		{Name: "/bcda/local/api/DATABASE_URL", Value: "postgresql://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable", Type: "SecureString"},
		{Name: "/bcda/local/api/QUEUE_DATABASE_URL", Value: "doesnt-matter", Type: "SecureString"},
	})
	defer cleanupParams()

	type test struct {
		path         string
		err          error
		expectedLogs []string
	}

	tests := []test{
		{path: "../../../shared_files/cclf/archives/valid2/", expectedLogs: []string{"Successfully imported 2 files.", "Failed to import 0 files.", "Skipped 1 files."}},
		{path: "../../../shared_files/cclf/archives/invalid_bcd/", err: errors.New("one or more files failed to import correctly"), expectedLogs: []string{"Successfully imported 0 files.", "Failed to import 1 files.", "Skipped 0 files."}},
		{path: "../../../shared_files/cclf/archives/skip/", expectedLogs: []string{"Successfully imported 0 files.", "Failed to import 0 files.", "Skipped 1 files."}},
	}

	for _, tc := range tests {
		postgrestest.DeleteCCLFFilesByCMSID(s.T(), database.Connection, targetACO)
		defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), database.Connection, targetACO)

		path, cleanup := testUtils.CopyToS3(s.T(), tc.path)
		defer cleanup()

		res, err := cclfImportHandler(context.Background(), testUtils.GetSQSEvent(s.T(), path, "fake_filename"))

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

func (s *CclfImportMainSuite) TestHandlerMissingS3AssumeRoleArn() {
	assert := assert.New(s.T())
	_, err := cclfImportHandler(context.Background(), testUtils.GetSQSEvent(s.T(), "doesn't-matter", "fake_filename"))
	assert.Contains(err.Error(), "Error retrieving parameter /cclf-import/bcda/local/bfd-bucket-role-arn from parameter store: ParameterNotFound: Parameter /cclf-import/bcda/local/bfd-bucket-role-arn not found.")
}
