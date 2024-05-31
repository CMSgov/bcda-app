package main

import (
	"testing"

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
		{Name: "/opt-out-import/bcda/local/bfd-s3-import-path", Value: path, Type: "String"},
		{Name: "/bcda/local/api/DATABASE_URL", Value: "postgresql://postgres:toor@db-unit-test:5432/bcda_test?sslmode=disable", Type: "SecureString"},
		{Name: "/bcda/local/api/QUEUE_DATABASE_URL", Value: "doesnt-matter", Type: "SecureString"},
	})
	defer cleanupParams()

	_, err := optOutImportHandler()
	assert.Nil(err)

	fs := postgrestest.GetSuppressionFileByName(s.T(), database.Connection,
		"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000010",
		"T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241391")

	assert.Len(fs, 2)

	for _, f := range fs {
		postgrestest.DeleteSuppressionFileByID(s.T(), database.Connection, f.ID)
	}
}

func (s *OptOutImportMainSuite) TestHandlerMissingS3AssumeRoleArn() {
	assert := assert.New(s.T())

	cleanupParams := testUtils.SetParameters(s.T(), []testUtils.AwsParameter{
		{Name: "/opt-out-import/bcda/local/bfd-s3-import-path", Value: "any-sort-of-path", Type: "String"},
	})
	defer cleanupParams()

	_, err := optOutImportHandler()
	assert.Contains(err.Error(), "invalid parameters error: /opt-out-import/bcda/local/bfd-bucket-role-arn")
}

func (s *OptOutImportMainSuite) TestHandlerMissingS3ImportPathKey() {
	assert := assert.New(s.T())

	cleanupParams := testUtils.SetParameters(s.T(), []testUtils.AwsParameter{
		{Name: "/opt-out-import/bcda/local/bfd-bucket-role-arn", Value: "arn:aws:iam::000000000000:user/fake-arn", Type: "String"},
	})
	defer cleanupParams()

	_, err := optOutImportHandler()
	assert.Contains(err.Error(), "invalid parameters error: /opt-out-import/bcda/local/bfd-s3-import-path")
}
