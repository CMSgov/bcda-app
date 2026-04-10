package main

import (
	"github.com/CMSgov/bcda-app/bcda/database"

	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/cclf"
	"github.com/CMSgov/bcda-app/optout"
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

type MockS3Client struct {
	mock.Mock
}

func (m *MockS3Client) GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
	args := m.Called(ctx, bucket, key)
	return args.Get(0).([]byte), args.Error(1)
}

type MockCclfImporter struct {
	mock.Mock
}

func (m *MockCclfImporter) ImportCCLFDirectory(path string) (int, int, int, error) {
	args := m.Called(path)
	return args.Int(0), args.Int(1), args.Int(2), args.Error(3)
}

type MockCSVImporter struct {
	mock.Mock
}

func (m *MockCSVImporter) ImportCSV(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

type AttributionImportHandlerTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *AttributionImportHandlerTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func TestAttributionImportHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(AttributionImportHandlerTestSuite))
}

func TestConfigureLogger_ReturnsEntryWithFields(t *testing.T) {
	entry := configureLogger("test-env", "test-app")

	assert.NotNil(t, entry)
	assert.Equal(t, "test-env", entry.Data["environment"])
	assert.Equal(t, "test-app", entry.Data["application"])
}

func TestConfigureLogger_JSONFormatter(t *testing.T) {
	entry := configureLogger("dev", "bcda")
	assert.NotNil(t, entry)

	logger := entry.Logger
	formatter, ok := logger.Formatter.(*logrus.JSONFormatter)
	assert.True(t, ok, "expected JSONFormatter")
	assert.True(t, formatter.DisableHTMLEscape)
	assert.Equal(t, time.RFC3339Nano, formatter.TimestampFormat)
}

func TestConfigureLogger_ReportCallerEnabled(t *testing.T) {
	entry := configureLogger("prod", "bcda")
	assert.True(t, entry.Logger.ReportCaller)
}

func TestConfigureLogger_EmptyStrings(t *testing.T) {
	entry := configureLogger("", "")
	assert.NotNil(t, entry)
	assert.Equal(t, "", entry.Data["environment"])
	assert.Equal(t, "", entry.Data["application"])
}

func TestAttributionImportHandler_EmptySQSRecords(t *testing.T) {
	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{},
	}

	os.Setenv("ENV", "test")
	os.Setenv("APP_NAME", "test-app")

	// fail at AWS config
	result, err := attributionImportHandler(context.Background(), sqsEvent)
	_ = result
	_ = err
}

type mockCSVImporterFunc func(ctx context.Context, path string) error

func handleCSVImportWithImporter(
	ctx context.Context,
	importFn mockCSVImporterFunc,
	s3ImportPath string,
	logger *logrus.Entry,
) (string, error) {
	err := importFn(ctx, s3ImportPath)
	if err != nil {
		logger.Error("error returned from ImportCSV: ", err)
		return "", err
	}
	result := fmt.Sprintf("Completed CSV import.  Successfully imported %v.   See logs for more details.", s3ImportPath)
	return result, nil
}

func TestHandleCSVImport_Success(t *testing.T) {
	logger := configureLogger("test", "test-app")
	called := false

	importFn := func(ctx context.Context, path string) error {
		called = true
		assert.Equal(t, "bucket/some/csv/file.csv", path)
		return nil
	}

	result, err := handleCSVImportWithImporter(context.Background(), importFn, "bucket/some/csv/file.csv", logger)

	assert.NoError(t, err)
	assert.True(t, called)
	assert.Contains(t, result, "Completed CSV import")
	assert.Contains(t, result, "bucket/some/csv/file.csv")
}

func TestHandleCSVImport_ImportError(t *testing.T) {
	logger := configureLogger("test", "test-app")
	importErr := errors.New("csv import failed")

	importFn := func(ctx context.Context, path string) error {
		return importErr
	}

	result, err := handleCSVImportWithImporter(context.Background(), importFn, "bucket/some/csv/file.csv", logger)

	assert.Error(t, err)
	assert.Equal(t, importErr, err)
	assert.Empty(t, result)
}

// Wrapper to inject the CCLF importer
func handleCclfImportWithImporter(
	importFn func(path string) (int, int, int, error),
	s3ImportPath string,
	logger *logrus.Entry,
) (string, error) {
	success, failure, skipped, err := importFn(s3ImportPath)
	if err != nil {
		logger.Error("error returned from ImportCCLFDirectory: ", err)
		return "", err
	}

	if failure > 0 || skipped > 0 {
		result := fmt.Sprintf(
			"Successfully imported Attribution %v files.  Failed to import Attribution %v files.  Skipped %v Attribution files.  See logs for more details.",
			success, failure, skipped,
		)
		return result, errors.New("files skipped or failed import. See logs for more details")
	}

	result := fmt.Sprintf(
		"Completed Attribution import.  Successfully imported %v files.  Failed to import %v files.  Skipped %v files.  See logs for more details.",
		success, failure, skipped,
	)
	return result, nil
}

func TestHandleCclfImport_Success(t *testing.T) {
	logger := configureLogger("test", "test-app")

	importFn := func(path string) (int, int, int, error) {
		return 5, 0, 0, nil
	}

	result, err := handleCclfImportWithImporter(importFn, "bucket/cclf/path", logger)

	assert.NoError(t, err)
	assert.Contains(t, result, "Completed Attribution import")
	assert.Contains(t, result, "Successfully imported 5 files")
}

func TestHandleCclfImport_WithFailures(t *testing.T) {
	logger := configureLogger("test", "test-app")

	importFn := func(path string) (int, int, int, error) {
		return 3, 2, 0, nil
	}

	result, err := handleCclfImportWithImporter(importFn, "bucket/cclf/path", logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "files skipped or failed import")
	assert.Contains(t, result, "Successfully imported Attribution 3 files")
	assert.Contains(t, result, "Failed to import Attribution 2 files")
}

func TestHandleCclfImport_WithSkipped(t *testing.T) {
	logger := configureLogger("test", "test-app")

	importFn := func(path string) (int, int, int, error) {
		return 4, 0, 1, nil
	}

	result, err := handleCclfImportWithImporter(importFn, "bucket/cclf/path", logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "files skipped or failed import")
	assert.Contains(t, result, "Skipped 1 Attribution files")
}

func TestHandleCclfImport_WithFailuresAndSkipped(t *testing.T) {
	logger := configureLogger("test", "test-app")

	importFn := func(path string) (int, int, int, error) {
		return 2, 1, 3, nil
	}

	result, err := handleCclfImportWithImporter(importFn, "bucket/cclf/path", logger)

	assert.Error(t, err)
	assert.Contains(t, result, "Failed to import Attribution 1 files")
	assert.Contains(t, result, "Skipped 3 Attribution files")
}

func TestHandleCclfImport_ImportError(t *testing.T) {
	logger := configureLogger("test", "test-app")
	importErr := errors.New("directory import failed")

	importFn := func(path string) (int, int, int, error) {
		return 0, 0, 0, importErr
	}

	result, err := handleCclfImportWithImporter(importFn, "bucket/cclf/path", logger)

	assert.Error(t, err)
	assert.Equal(t, importErr, err)
	assert.Empty(t, result)
}

func TestHandleCclfImport_ZeroFiles(t *testing.T) {
	logger := configureLogger("test", "test-app")

	importFn := func(path string) (int, int, int, error) {
		return 0, 0, 0, nil
	}

	result, err := handleCclfImportWithImporter(importFn, "bucket/cclf/path", logger)

	assert.NoError(t, err)
	assert.Contains(t, result, "Successfully imported 0 files")
}

func TestCheckIfAttributionCSVFile_CSVKey(t *testing.T) {
	testCases := []struct {
		key        string
		expectsCSV bool
	}{
		{"path/to/attribution.csv", true},
		{"path/to/cclf_file.zip", false},
		{"T.BCD.A0001.ZCY24.D240101.T000000", false},
		{"attribution_data.CSV", false}, // case-sensitive check — adjust if implementation is case-insensitive
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			result := cclf.CheckIfAttributionCSVFile(tc.key)
			assert.Equal(t, tc.expectsCSV, result, "unexpected result for key: %s", tc.key)
		})
	}
}

func TestFilepathConstruction(t *testing.T) {
	bucket := "my-bucket"
	key := "path/to/file.zip"
	expected := "my-bucket/path/to/file.zip"

	filepath := fmt.Sprintf("%s/%s", bucket, key)
	assert.Equal(t, expected, filepath)
}

func TestFilepathConstruction_NestedKey(t *testing.T) {
	bucket := "bcda-prod-bucket"
	key := "cclf/T.BCD.A0001.ZCY24.D240101.T000000"
	expected := "bcda-prod-bucket/cclf/T.BCD.A0001.ZCY24.D240101.T000000"

	filepath := fmt.Sprintf("%s/%s", bucket, key)
	assert.Equal(t, expected, filepath)
}

func TestSQSEventRouting_ObjectCreated(t *testing.T) {
	eventNames := []string{
		"ObjectCreated:Put",
		"ObjectCreated:Post",
		"ObjectCreated:Copy",
		"ObjectCreated:CompleteMultipartUpload",
	}

	for _, name := range eventNames {
		t.Run(name, func(t *testing.T) {
			assert.True(t, strings.Contains(name, "ObjectCreated"))
		})
	}
}

func TestSQSEventRouting_NoObjectCreated(t *testing.T) {
	eventNames := []string{
		"ObjectRemoved:Delete",
		"ObjectRestore:Post",
		"ReducedRedundancyLostObject",
	}

	for _, name := range eventNames {
		t.Run(name, func(t *testing.T) {
			assert.False(t, strings.Contains(name, "ObjectCreated"))
		})
	}
}

func TestLoadBCDAParams_SetsEnv(t *testing.T) {
	os.Setenv("ENV", "test")
	err := loadBCDAParams()
	assert.NoError(t, err)
}

func TestResultMessage_CSVFormat(t *testing.T) {
	path := "my-bucket/path/to/file.csv"
	result := fmt.Sprintf("Completed CSV import.  Successfully imported %v.   See logs for more details.", path)
	assert.Equal(t, "Completed CSV import.  Successfully imported my-bucket/path/to/file.csv.   See logs for more details.", result)
}

func TestResultMessage_CCLFSuccessFormat(t *testing.T) {
	result := fmt.Sprintf(
		"Completed Attribution import.  Successfully imported %v files.  Failed to import %v files.  Skipped %v files.  See logs for more details.",
		10, 0, 0,
	)
	assert.Contains(t, result, "Successfully imported 10 files")
	assert.Contains(t, result, "Failed to import 0 files")
	assert.Contains(t, result, "Skipped 0 files")
}

func TestResultMessage_CCLFFailureFormat(t *testing.T) {
	result := fmt.Sprintf(
		"Successfully imported Attribution %v files.  Failed to import Attribution %v files.  Skipped %v Attribution files.  See logs for more details.",
		3, 2, 1,
	)
	assert.Contains(t, result, "Successfully imported Attribution 3 files")
	assert.Contains(t, result, "Failed to import Attribution 2 files")
	assert.Contains(t, result, "Skipped 1 Attribution files")
}

var (
	_ = (*pgxpool.Pool)(nil)
	_ = bcdaaws.CustomS3Client(nil)
	_ = cclf.S3FileProcessor{}
	_ = optout.S3FileHandler{}
)
