package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/database"
)

var (
	testapp    = "test-app"
	bucketcsv  = "bucket/some/csv/file.csv"
	bucketcclf = "bucket/cclf/path"
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

func TestHandleCSVImportNoACOConfig(t *testing.T) {
	s3Client := &bcdaaws.MockS3Client{}
  	pool := database.ConnectPool()

	_, err := handleCSVImport(context.Background(), pool, s3Client, "../../../sharedfiles/csv/valid.csv")
	assert.ErrorContains(t, err, "CSV Attribution metadata invalid: No ACO configs found")
}

func TestConfigureLogger(t *testing.T) {
	t.Run("fields are populated", func(t *testing.T) {
		entry := configureLogger("test-env", testapp)
		require.NotNil(t, entry)
		assert.Equal(t, "test-env", entry.Data["environment"])
		assert.Equal(t, testapp, entry.Data["application"])
	})

	t.Run("uses JSON formatter with correct settings", func(t *testing.T) {
		entry := configureLogger("dev", "bcda")
		require.NotNil(t, entry)
		formatter, ok := entry.Logger.Formatter.(*logrus.JSONFormatter)
		require.True(t, ok, "expected *logrus.JSONFormatter")
		assert.True(t, formatter.DisableHTMLEscape)
		assert.Equal(t, time.RFC3339Nano, formatter.TimestampFormat)
	})

	t.Run("ReportCaller is enabled", func(t *testing.T) {
		entry := configureLogger("prod", "bcda")
		assert.True(t, entry.Logger.ReportCaller)
	})

	t.Run("accepts empty strings", func(t *testing.T) {
		entry := configureLogger("", "")
		require.NotNil(t, entry)
		assert.Equal(t, "", entry.Data["environment"])
		assert.Equal(t, "", entry.Data["application"])
	})
}

type csvImportFunc func(ctx context.Context, path string) error

func handleCSVImportWithImporter(
	ctx context.Context,
	importFn csvImportFunc,
	s3ImportPath string,
	logger *logrus.Entry,
) (string, error) {
	if err := importFn(ctx, s3ImportPath); err != nil {
		logger.Error("error returned from ImportCSV: ", err)
		return "", err
	}
	return fmt.Sprintf(
		"Completed CSV import.  Successfully imported %v.   See logs for more details.",
		s3ImportPath,
	), nil
}

func TestHandleCSVImport(t *testing.T) {
	logger := configureLogger("test", testapp)

	t.Run("success — returns result containing path", func(t *testing.T) {
		called := false
		importFn := func(ctx context.Context, path string) error {
			called = true
			assert.Equal(t, bucketcsv, path)
			return nil
		}

		result, err := handleCSVImportWithImporter(context.Background(), importFn, bucketcsv, logger)

		require.NoError(t, err)
		assert.True(t, called, "import function was never called")
		assert.Contains(t, result, "Completed CSV import")
		assert.Contains(t, result, bucketcsv)
	})

	t.Run("importer error — propagates error and returns empty result", func(t *testing.T) {
		importErr := errors.New("csv import failed")
		importFn := func(_ context.Context, _ string) error { return importErr }

		result, err := handleCSVImportWithImporter(context.Background(), importFn, bucketcsv, logger)

		require.ErrorIs(t, err, importErr)
		assert.Empty(t, result)
	})
}

type cclfImportFunc func(path string) (success, failure, skipped int, err error)

func handleCclfImportWithImporter(
	importFn cclfImportFunc,
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

	return fmt.Sprintf(
		"Completed Attribution import.  Successfully imported %v files.  Failed to import %v files.  Skipped %v files.  See logs for more details.",
		success, failure, skipped,
	), nil
}

func TestHandleCclfImport(t *testing.T) {
	logger := configureLogger("test", testapp)

	t.Run("all files succeed", func(t *testing.T) {
		result, err := handleCclfImportWithImporter(
			func(_ string) (int, int, int, error) { return 5, 0, 0, nil },
			bucketcclf, logger,
		)
		require.NoError(t, err)
		assert.Contains(t, result, "Completed Attribution import")
		assert.Contains(t, result, "Successfully imported 5 files")
	})

	t.Run("zero files — treated as success", func(t *testing.T) {
		result, err := handleCclfImportWithImporter(
			func(_ string) (int, int, int, error) { return 0, 0, 0, nil },
			bucketcclf, logger,
		)
		require.NoError(t, err)
		assert.Contains(t, result, "Successfully imported 0 files")
	})

	partialFailureCases := []struct {
		name                      string
		success, failure, skipped int
		wantResultParts           []string
	}{
		{
			name:    "some files fail",
			success: 3, failure: 2, skipped: 0,
			wantResultParts: []string{
				"Successfully imported Attribution 3 files",
				"Failed to import Attribution 2 files",
			},
		},
		{
			name:    "some files skipped",
			success: 4, failure: 0, skipped: 1,
			wantResultParts: []string{"Skipped 1 Attribution files"},
		},
		{
			name:    "failures and skips combined",
			success: 2, failure: 1, skipped: 3,
			wantResultParts: []string{
				"Failed to import Attribution 1 files",
				"Skipped 3 Attribution files",
			},
		},
	}

	for _, tc := range partialFailureCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			s, f, sk := tc.success, tc.failure, tc.skipped
			result, err := handleCclfImportWithImporter(
				func(_ string) (int, int, int, error) { return s, f, sk, nil },
				bucketcclf, logger,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "files skipped or failed import")
			for _, part := range tc.wantResultParts {
				assert.Contains(t, result, part)
			}
		})
	}

	t.Run("importer error — propagates error and returns empty result", func(t *testing.T) {
		importErr := errors.New("directory import failed")
		result, err := handleCclfImportWithImporter(
			func(_ string) (int, int, int, error) { return 0, 0, 0, importErr },
			bucketcclf, logger,
		)
		require.ErrorIs(t, err, importErr)
		assert.Empty(t, result)
	})
}

func TestLoadBCDAParams(t *testing.T) {
	t.Run("returns no error when ENV is set", func(t *testing.T) {
		t.Setenv("ENV", "test")
		require.NoError(t, loadBCDAParams())
	})

	t.Run("ENV value is propagated after load", func(t *testing.T) {
		t.Setenv("ENV", "staging")
		require.NoError(t, loadBCDAParams())
		assert.Equal(t, "staging", os.Getenv("ENV"))
	})
}
