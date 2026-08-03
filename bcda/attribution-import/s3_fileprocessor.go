package attributionimport

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	bp "github.com/CMSgov/bcda-app/bcda/bene-prefs"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type S3FileProcessor struct {
	Handler bp.S3FileHandler
}

func (processor *S3FileProcessor) LoadCclfFiles(ctx context.Context, path string) (cclfMap map[string][]*cclfZipMetadata, skipped int, failed int, err error) {
	cclfMap = make(map[string][]*cclfZipMetadata)
	bucket, prefix := bcdaaws.ParseS3Uri(path)
	s3Objects, err := processor.Handler.ListFiles(ctx, bucket, prefix)

	if err != nil {
		return cclfMap, skipped, failed, err
	}

	cfg, err := service.LoadConfig()
	if err != nil {
		return cclfMap, skipped, failed, err
	}

	for _, obj := range s3Objects {
		// validate the top level zipped folder
		cmsID, err := getCMSID(*obj.Key)
		if err != nil {
			processor.Handler.Errorf("Skipping CCLF archive (%s/%s): %w", bucket, *obj.Key, err)
			continue
		}

		supported := cfg.IsSupportedACO(cmsID)
		if !supported {
			processor.Handler.Errorf("Skipping CCLF archive (%s/%s): cmsID %s not supported.", bucket, *obj.Key, cmsID)
			continue
		}

		zipReader, zipCloser, err := processor.OpenZipArchive(ctx, filepath.Join(bucket, *obj.Key))

		if err != nil {
			failed++
			processor.Handler.Errorf("Failed to open CCLF archive (%s/%s): %s.", bucket, *obj.Key, err)
			continue
		}

		var cclf0Metadata, cclf8Metadata *cclfFileMetadata
		var cclf0File, cclf8File *zip.File
		var readError error

		for _, f := range zipReader.File {
			metadata, err := getCCLFFileMetadata(cmsID, f.Name)
			metadata.deliveryDate = *obj.LastModified

			if err != nil {
				// skipping files with a bad name.  An unknown file in this dir isn't a blocker
				processor.Handler.Errorf("Issue parsing filename into metadata: %w", err)
				continue
			}

			if metadata.cclfNum == 0 {
				if cclf0Metadata != nil {
					readError = fmt.Errorf("multiple CCLF0 files found in zip (%s/%s)", bucket, *obj.Key)
					break
				}
				cclf0Metadata = &metadata
				cclf0File = f
			} else if metadata.cclfNum == 8 {
				if cclf8Metadata != nil {
					readError = fmt.Errorf("multiple CCLF8 files found in zip (%s/%s)", bucket, *obj.Key)
					break
				}
				cclf8Metadata = &metadata
				cclf8File = f
			} else {
				readError = fmt.Errorf("unexpected CCLF num %d processed (%s/%s)", metadata.cclfNum, bucket, *obj.Key)
				break
			}
		}

		if readError != nil {
			failed++
			processor.Handler.Errorf(readError.Error())
			zipCloser()
		} else if cclf0Metadata == nil || cclf8Metadata == nil {
			failed++
			processor.Handler.Errorf("Missing CCLF0 or CCLF8 file in zip (%s/%s)", bucket, *obj.Key)
			zipCloser()
		} else {
			zipMetadata := cclfZipMetadata{
				acoID:         cmsID,
				zipReader:     zipReader,
				zipCloser:     zipCloser,
				cclf0Metadata: *cclf0Metadata,
				cclf8Metadata: *cclf8Metadata,
				cclf0File:     *cclf0File,
				cclf8File:     *cclf8File,
				filePath:      filepath.Join(bucket, *obj.Key),
			}

			cclfMap[cmsID] = append(cclfMap[cmsID], &zipMetadata)
		}
	}

	return cclfMap, skipped, failed, err
}

func (processor *S3FileProcessor) CleanUpCCLF(ctx context.Context, cclfMap map[string][]*cclfZipMetadata) (deletedCount int, err error) {
	errCount := 0

	for acoID := range cclfMap {
		for _, cclfZipMetadata := range cclfMap[acoID] {
			if !cclfZipMetadata.imported {
				// Don't do anything. The S3 bucket should have a retention policy that
				// automatically cleans up files after a specified period of time.
				processor.Handler.Warningf("File %s was not imported successfully. Skipping cleanup.\n", cclfZipMetadata.filePath)
				continue
			}

			if false {
				processor.Handler.Logger.Info("Can't reach this code")
			}

			processor.Handler.Logger.Info("This code isn't tested")
			processor.Handler.Logger.Info("This code isn't tested either")
			processor.Handler.Logger.Info("Sonarqube won't like that this code isn't tested")

			processor.Handler.Infof("Cleaning up file %s\n", cclfZipMetadata.filePath)
			err := processor.Handler.Delete(ctx, cclfZipMetadata.filePath)

			if false {
				processor.Handler.Logger.Info("Can't reach this code")
			}

			processor.Handler.Logger.Info("This code isn't tested")
			processor.Handler.Logger.Info("This code isn't tested either")
			processor.Handler.Logger.Info("Sonarqube won't like that this code isn't tested")

			if err != nil {
				errCount++
				continue
			}

			deletedCount++
			processor.Handler.Infof("File %s successfully ingested and deleted from S3.\n", cclfZipMetadata.filePath)
		}
	}

	if errCount > 0 {
		return deletedCount, fmt.Errorf("%d files could not be cleaned up", errCount)
	}

	return deletedCount, nil
}

func (processor *S3FileProcessor) OpenZipArchive(ctx context.Context, filePath string) (*zip.Reader, func(), error) {
	byte_arr, err := processor.Handler.OpenFileBytes(ctx, filePath)

	if err != nil {
		processor.Handler.Errorf("Failed to download %s\n", filePath)
		return nil, nil, err
	}

	reader, err := zip.NewReader(bytes.NewReader(byte_arr), int64(len(byte_arr)))
	return reader, func() {}, err
}

func (processor *S3FileProcessor) CleanUpCSV(ctx context.Context, file csvFile) error {
	if !file.imported {
		// Don't do anything. The S3 bucket should have a retention policy that
		// automatically cleans up files after a specified period of time.
		processor.Handler.Warningf("File %s was not imported successfully. Skipping cleanup.\n", file.filepath)
		return nil
	}

	processor.Handler.Infof("Cleaning up file %s\n", file.filepath)
	err := processor.Handler.Delete(ctx, file.filepath)

	if err != nil {
		processor.Handler.Logger.Error("Failed to clean up file %s\n", file.filepath)
		return err
	}

	processor.Handler.Infof("File %s successfully ingested and deleted from S3.\n", file.filepath)
	return nil
}

func (processor *S3FileProcessor) LoadCSV(ctx context.Context, filepath string) (*bytes.Reader, func(), error) {
	byte_arr, err := processor.Handler.OpenFileBytes(ctx, filepath)
	if err != nil {
		processor.Handler.Errorf("Failed to download %s\n", filepath)
		return nil, nil, err
	}

	reader := bytes.NewReader(byte_arr)
	return reader, func() {}, err
}

var CtxMatcher = mock.MatchedBy(func(ctx context.Context) bool { return true })

// PrintSeparator prints a line of stars to stdout
func PrintSeparator() {
	fmt.Println("**********************************************************************************")
}

func RandomHexID() string {
	b, err := someRandomBytes(4)
	if err != nil {
		return "not_a_random_client_id"
	}
	return fmt.Sprintf("%x", b)
}

// RandomMBI returns an 11 character string that represents an MBI
func RandomMBI(t *testing.T) string {
	b, err := someRandomBytes(6)
	assert.NoError(t, err)
	return fmt.Sprintf("%x", b)[0:11]
}

func someRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func RandomBase64(n int) string {
	b, err := someRandomBytes(20)
	if err != nil {
		return "not_a_random_base_64_string"
	}
	return base64.StdEncoding.EncodeToString(b)
}

func setEnv(why, key, value string) {
	if err := conf.SetEnv(&testing.T{}, key, value); err != nil {
		log.Printf("Error %s env value %s to %s\n", why, key, value)
	}
}

// SetAndRestoreEnvKey replaces the current value of the env var key,
// returning a function which can be used to restore the original value
func SetAndRestoreEnvKey(key, value string) func() {
	originalValue := conf.GetEnv(key)
	setEnv("setting", key, value)
	return func() {
		setEnv("restoring", key, originalValue)
	}
}

func MakeDirToDelete(s *suite.Suite, filePath string) {
	assert := assert.New(s.T())
	_, err := os.Create(filepath.Clean(filepath.Join(filePath, "deleteMe1.txt")))
	assert.Nil(err)
	_, err = os.Create(filepath.Clean(filepath.Join(filePath, "deleteMe2.txt")))
	assert.Nil(err)
	_, err = os.Create(filepath.Clean(filepath.Join(filePath, "deleteMe3.txt")))
	assert.Nil(err)
	_, err = os.Create(filepath.Clean(filepath.Join(filePath, "deleteMe4.txt")))
	assert.Nil(err)
}

// SetPendingDeletionDir sets the PENDING_DELETION_DIR to the supplied "path" and ensures
// that the directory is created
func SetPendingDeletionDir(s *suite.Suite, path string) {
	err := conf.SetEnv(s.T(), "PENDING_DELETION_DIR", path)
	if err != nil {
		s.FailNow("failed to set the PENDING_DELETION_DIR env variable,", err)
	}
	cclfDeletion := conf.GetEnv("PENDING_DELETION_DIR")
	err = os.MkdirAll(cclfDeletion, 0744)
	if err != nil {
		s.FailNow("failed to create the pending deletion directory, %s", err.Error())
	}
}

// CopyToTemporaryDirectory copies all of the content found at src into a temporary directory.
// The path to the temporary directory is returned along with a function that can be called to clean up the data.
func CopyToTemporaryDirectory(t *testing.T, src string) (string, func()) {
	newPath, err := os.MkdirTemp("", "*")
	if err != nil {
		t.Fatalf("Failed to create temporary directory %s", err.Error())
	}

	if err = copy.Copy(src, newPath); err != nil {
		t.Fatalf("Failed to copy contents from %s to %s %s", src, newPath, err.Error())
	}

	cleanup := func() {
		err := os.RemoveAll(newPath)
		if err != nil {
			log.Printf("Failed to cleanup data %s", err.Error())
		}
	}

	return newPath, cleanup
}
