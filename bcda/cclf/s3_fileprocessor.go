package cclf

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/CMSgov/bcda-app/bcda/cclf/metrics"
	ers "github.com/CMSgov/bcda-app/bcda/errors"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/optout"
)

type S3FileProcessor struct {
	Handler optout.S3FileHandler
}

func (processor *S3FileProcessor) LoadCclfFiles(path string) (cclfMap map[string][]*cclfZipMetadata, skipped int, failed int, err error) {
	cclfMap = make(map[string][]*cclfZipMetadata)
	bucket, prefix := optout.ParseS3Uri(path)
	s3Objects, err := processor.Handler.ListFiles(bucket, prefix)

	if err != nil {
		return cclfMap, skipped, failed, err
	}

	for _, obj := range s3Objects {
		// ignore the opt out file, and don't add it to the skipped count
		optOut, _ := optout.IsOptOut(*obj.Key)
		if optOut {
			processor.Handler.Infof("Skipping opt-out file: %s/%s", bucket, *obj.Key)
			continue
		}

		// ignore files for different environments, and don't add it to the skipped count
		if !optout.IsForCurrentEnv(*obj.Key) {
			processor.Handler.Infof("Skipping file for different environment: %s/%s", bucket, *obj.Key)
			continue
		}

		// validate the top level zipped folder
		cmsID, err := getCMSID(*obj.Key)
		if err != nil {
			processor.Handler.Warningf("Skipping CCLF archive (%s/%s): %s.", bucket, *obj.Key, err)
			continue
		}

		supported := service.IsSupportedACO(cmsID)
		if !supported {
			processor.Handler.Errorf("Skipping CCLF archive (%s/%s): cmsID %s not supported.", bucket, *obj.Key, cmsID)
			continue
		}

		zipReader, zipCloser, err := processor.OpenZipArchive(filepath.Join(bucket, *obj.Key))

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
				processor.Handler.Warningf("Unknown file found: %s.", f.Name)
				continue
			}

			if metadata.cclfNum == 0 {
				if cclf0Metadata != nil {
					readError = fmt.Errorf("Multiple CCLF0 files found in zip (%s/%s)", bucket, *obj.Key)
					break
				}
				cclf0Metadata = &metadata
				cclf0File = f
			} else if metadata.cclfNum == 8 {
				if cclf8Metadata != nil {
					readError = fmt.Errorf("Multiple CCLF8 files found in zip (%s/%s)", bucket, *obj.Key)
					break
				}
				cclf8Metadata = &metadata
				cclf8File = f
			} else {
				readError = fmt.Errorf("Unexpected CCLF num %d processed (%s/%s)", metadata.cclfNum, bucket, *obj.Key)
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
			close := metrics.NewChild(ctx, "cleanUpCCLFZip")
			defer close()

			if !cclfZipMetadata.imported {
				// Don't do anything. The S3 bucket should have a retention policy that
				// automatically cleans up files after a specified period of time.
				processor.Handler.Warningf("File %s was not imported successfully. Skipping cleanup.\n", cclfZipMetadata.filePath)
				continue
			}

			processor.Handler.Infof("Cleaning up file %s\n", cclfZipMetadata.filePath)
			err := processor.Handler.Delete(cclfZipMetadata.filePath)

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

func (processor *S3FileProcessor) OpenZipArchive(filePath string) (*zip.Reader, func(), error) {
	byte_arr, err := processor.Handler.OpenFileBytes(filePath)

	if err != nil {
		processor.Handler.Errorf("Failed to download %s\n", filePath)
		return nil, nil, err
	}

	reader, err := zip.NewReader(bytes.NewReader(byte_arr), int64(len(byte_arr)))
	return reader, func() {}, err
}

func (processor *S3FileProcessor) CleanUpCSV(file csvFile) error {

	close := metrics.NewChild(context.Background(), "cleanUpCCLFZip")
	defer close()

	if !file.imported {
		// Don't do anything. The S3 bucket should have a retention policy that
		// automatically cleans up files after a specified period of time.
		processor.Handler.Warningf("File %s was not imported successfully. Skipping cleanup.\n", file.filepath)
		return nil
	}

	processor.Handler.Infof("Cleaning up file %s\n", file.filepath)
	err := processor.Handler.Delete(file.filepath)

	if err != nil {
		processor.Handler.Logger.Error("Failed to clean up file %s\n", file.filepath)
		return err

	}

	processor.Handler.Infof("File %s successfully ingested and deleted from S3.\n", file.filepath)
	return err
}

func (processor *S3FileProcessor) LoadCSV(filepath string) (*bytes.Reader, func(), error) {
	if !optout.IsForCurrentEnv(filepath) {
		processor.Handler.Infof("Skipping file for different environment: %s/%s", filepath)
		return nil, nil, &ers.AttributionFileMismatchedEnv{}
	}
	byte_arr, err := processor.Handler.OpenFileBytes(filepath)
	if err != nil {
		processor.Handler.Errorf("Failed to download %s\n", filepath)
		return nil, nil, err
	}

	reader := bytes.NewReader(byte_arr)
	return reader, func() {}, err
}
