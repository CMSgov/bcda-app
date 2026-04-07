package main

import (
	"context"
	"database/sql"
	"testing"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/database"
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

func TestHandleCSVImport_NoACOConfig(t *testing.T) {
	s3Client := &bcdaaws.MockS3Client{}
	pool := database.ConnectPool()

	path := "../../../shared_files/csv/valid.csv"

	_, err := handleCSVImport(context.Background(), pool, s3Client, path)
	assert.ErrorContains(t, err, "CSV Attribution metadata invalid: No ACO configs found")
}
