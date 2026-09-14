package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
)

var (
	testapp    = "test-app"
	bucketcsv  = "bucket/some/csv/file.csv"
	bucketcclf = "bucket/cclf/path"
)

type mockCSVImporter struct {
	importCSVFn func(ctx context.Context, filepath string) error
}

func (m *mockCSVImporter) ImportCSV(ctx context.Context, filepath string) error {
	if m.importCSVFn != nil {
		return m.importCSVFn(ctx, filepath)
	}
	return nil
}

type mockCCLFImporter struct {
	importCCLFDirectoryFn func(ctx context.Context, filePath string) (success, failure, skipped int, err error)
}

func (m *mockCCLFImporter) ImportCCLFDirectory(ctx context.Context, filePath string) (success, failure, skipped int, err error) {
	if m.importCCLFDirectoryFn != nil {
		return m.importCCLFDirectoryFn(ctx, filePath)
	}
	return 0, 0, 0, nil
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

func TestHandleCSVImport(t *testing.T) {
	logger := configureLogger("test", testapp)

	t.Run("success — returns result containing path", func(t *testing.T) {
		called := false
		importer := &mockCSVImporter{
			importCSVFn: func(ctx context.Context, path string) error {
				called = true
				assert.Equal(t, bucketcsv, path)
				return nil
			},
		}

		handler := &AttributionImportHandler{
			logger:      logger,
			csvImporter: importer,
		}

		result, err := handler.handleCSVImport(context.Background(), bucketcsv)

		require.NoError(t, err)
		assert.True(t, called, "import function was never called")
		assert.Contains(t, result, "Completed CSV import")
		assert.Contains(t, result, bucketcsv)
	})

	t.Run("importer error — propagates error and returns empty result", func(t *testing.T) {
		importErr := errors.New("csv import failed")
		importer := &mockCSVImporter{
			importCSVFn: func(_ context.Context, _ string) error { return importErr },
		}

		handler := &AttributionImportHandler{
			logger:      logger,
			csvImporter: importer,
		}

		result, err := handler.handleCSVImport(context.Background(), bucketcsv)

		require.ErrorIs(t, err, importErr)
		assert.Empty(t, result)
	})
}

func TestHandleCclfImport(t *testing.T) {
	logger := configureLogger("test", testapp)

	t.Run("all files succeed", func(t *testing.T) {
		importer := &mockCCLFImporter{
			importCCLFDirectoryFn: func(_ context.Context, _ string) (int, int, int, error) {
				return 5, 0, 0, nil
			},
		}
		handler := &AttributionImportHandler{
			logger:       logger,
			cclfImporter: importer,
		}

		result, err := handler.handleCclfImport(context.Background(), bucketcclf)
		require.NoError(t, err)
		assert.Contains(t, result, "Completed Attribution import")
		assert.Contains(t, result, "Successfully imported 5 files")
	})

	t.Run("zero files — treated as success", func(t *testing.T) {
		importer := &mockCCLFImporter{
			importCCLFDirectoryFn: func(_ context.Context, _ string) (int, int, int, error) {
				return 0, 0, 0, nil
			},
		}
		handler := &AttributionImportHandler{
			logger:       logger,
			cclfImporter: importer,
		}

		result, err := handler.handleCclfImport(context.Background(), bucketcclf)
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
			importer := &mockCCLFImporter{
				importCCLFDirectoryFn: func(_ context.Context, _ string) (int, int, int, error) {
					return s, f, sk, nil
				},
			}
			handler := &AttributionImportHandler{
				logger:       logger,
				cclfImporter: importer,
			}

			result, err := handler.handleCclfImport(context.Background(), bucketcclf)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "files skipped or failed import")
			for _, part := range tc.wantResultParts {
				assert.Contains(t, result, part)
			}
		})
	}

	t.Run("importer error — propagates error and returns empty result", func(t *testing.T) {
		importErr := errors.New("directory import failed")
		importer := &mockCCLFImporter{
			importCCLFDirectoryFn: func(_ context.Context, _ string) (int, int, int, error) {
				return 0, 0, 0, importErr
			},
		}
		handler := &AttributionImportHandler{
			logger:       logger,
			cclfImporter: importer,
		}

		result, err := handler.handleCclfImport(context.Background(), bucketcclf)
		require.ErrorIs(t, err, importErr)
		assert.Empty(t, result)
	})
}

func TestHandleSQSEvent(t *testing.T) {
	logger := configureLogger("test", testapp)

	t.Run("empty sqs event returns safely", func(t *testing.T) {
		handler := &AttributionImportHandler{logger: logger}
		result, err := handler.handleImport(context.Background(), events.SQSEvent{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("routes CSV file correctly", func(t *testing.T) {
		called := false
		csvImporter := &mockCSVImporter{
			importCSVFn: func(_ context.Context, path string) error {
				called = true
				assert.Equal(t, "test-bucket/cclf/archives/csv/P.PCPB.M2411.D181120.T1000000", path)
				return nil
			},
		}
		handler := &AttributionImportHandler{
			logger:      logger,
			csvImporter: csvImporter,
			checkIfCSV: func(filePath string) (bool, error) {
				return true, nil
			},
		}

		sqsEvent := testUtils.GetSQSEvent(t, "test-bucket", "cclf/archives/csv/P.PCPB.M2411.D181120.T1000000")
		result, err := handler.handleImport(context.Background(), sqsEvent)
		require.NoError(t, err)
		assert.True(t, called)
		assert.Contains(t, result, "Completed CSV import")
	})

	t.Run("routes CCLF zip file correctly", func(t *testing.T) {
		called := false
		cclfImporter := &mockCCLFImporter{
			importCCLFDirectoryFn: func(_ context.Context, path string) (int, int, int, error) {
				called = true
				assert.Equal(t, "test-bucket/cclf/archives/valid/T.BCD.A0001.ZCY18.D181120.T1000000", path)
				return 1, 0, 0, nil
			},
		}
		handler := &AttributionImportHandler{
			logger:       logger,
			cclfImporter: cclfImporter,
			checkIfCSV: func(filePath string) (bool, error) {
				return false, nil
			},
		}

		sqsEvent := testUtils.GetSQSEvent(t, "test-bucket", "cclf/archives/valid/T.BCD.A0001.ZCY18.D181120.T1000000")
		result, err := handler.handleImport(context.Background(), sqsEvent)
		require.NoError(t, err)
		assert.True(t, called)
		assert.Contains(t, result, "Completed Attribution import")
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
