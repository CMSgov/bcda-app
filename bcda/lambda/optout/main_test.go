package main

import (
	"context"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type OptOutImportMainSuite struct {
	suite.Suite
}

func TestOptOutImportMainSuite(t *testing.T) {
	suite.Run(t, new(OptOutImportMainSuite))
}

func (s *OptOutImportMainSuite) TestOptOutImportHandlerSuccess() {
	assert := assert.New(s.T())
	path, cleanup := testUtils.CopyToS3(s.T(), "../../../shared_files/synthetic1800MedicareFiles/test2/")
	defer cleanup()

	cleanupParams := testUtils.SetParameters(s.T(), []testUtils.AwsParameter{
		{Name: "/opt-out-import/bcda/local/bfd-bucket-role-arn", Value: "arn:aws:iam::000000000000:user/fake-arn", Type: "String"},
		{Name: "/bcda/local/api/DATABASE_URL", Value: "postgresql://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable", Type: "SecureString"},
		{Name: "/bcda/local/api/QUEUE_DATABASE_URL", Value: "doesnt-matter", Type: "SecureString"},
	})
	defer cleanupParams()

	res, err := optOutImportHandler(context.Background(), testUtils.GetSQSEvent(s.T(), path, "fake_filename"))
	assert.Nil(err)
	assert.Contains(res, constants.CompleteMedSupDataImp)
	assert.Contains(res, "Files imported: 2")
	assert.Contains(res, "Files failed: 0")
	assert.Contains(res, "Files skipped: 0")

	fs := postgrestest.GetSuppressionFileByName(s.T(), database.Connection,
		"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000010",
		"T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241391")

	assert.Len(fs, 2)

	for _, f := range fs {
		postgrestest.DeleteSuppressionFileByID(s.T(), database.Connection, f.ID)
	}
}

func (s *OptOutImportMainSuite) TestImportSuppressionDirectory_Skipped() {
	assert := assert.New(s.T())
	path, cleanup := testUtils.CopyToS3(s.T(), "../../../shared_files/suppressionfile_BadFileNames/")
	defer cleanup()

	cleanupParams := testUtils.SetParameters(s.T(), []testUtils.AwsParameter{
		{Name: "/opt-out-import/bcda/local/bfd-bucket-role-arn", Value: "arn:aws:iam::000000000000:user/fake-arn", Type: "String"},
		{Name: "/bcda/local/api/DATABASE_URL", Value: "postgresql://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable", Type: "SecureString"},
		{Name: "/bcda/local/api/QUEUE_DATABASE_URL", Value: "doesnt-matter", Type: "SecureString"},
	})
	defer cleanupParams()

	res, err := optOutImportHandler(context.Background(), testUtils.GetSQSEvent(s.T(), path, "fake_filename"))
	assert.Nil(err)
	assert.Contains(res, constants.CompleteMedSupDataImp)
	assert.Contains(res, "Files imported: 0")
	assert.Contains(res, "Files failed: 0")
	assert.Contains(res, "Files skipped: 2")
}

func (s *OptOutImportMainSuite) TestImportSuppressionDirectory_Failed() {
	assert := assert.New(s.T())
	path, cleanup := testUtils.CopyToS3(s.T(), "../../../shared_files/suppressionfile_BadHeader/")
	defer cleanup()

	cleanupParams := testUtils.SetParameters(s.T(), []testUtils.AwsParameter{
		{Name: "/opt-out-import/bcda/local/bfd-bucket-role-arn", Value: "arn:aws:iam::000000000000:user/fake-arn", Type: "String"},
		{Name: "/bcda/local/api/DATABASE_URL", Value: "postgresql://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable", Type: "SecureString"},
		{Name: "/bcda/local/api/QUEUE_DATABASE_URL", Value: "doesnt-matter", Type: "SecureString"},
	})
	defer cleanupParams()

	res, err := optOutImportHandler(context.Background(), testUtils.GetSQSEvent(s.T(), path, "fake_filename"))
	assert.EqualError(err, "one or more suppression files failed to import correctly")
	assert.Contains(res, constants.CompleteMedSupDataImp)
	assert.Contains(res, "Files imported: 0")
	assert.Contains(res, "Files failed: 1")
	assert.Contains(res, "Files skipped: 0")
}

func (s *OptOutImportMainSuite) TestHandlerMissingS3AssumeRoleArn() {
	assert := assert.New(s.T())
	_, err := optOutImportHandler(context.Background(), testUtils.GetSQSEvent(s.T(), "doesn't-matter", "fake_filename"))
	assert.Contains(err.Error(), "Error retrieving parameter /opt-out-import/bcda/local/bfd-bucket-role-arn from parameter store: ParameterNotFound: Parameter /opt-out-import/bcda/local/bfd-bucket-role-arn not found.")
}
