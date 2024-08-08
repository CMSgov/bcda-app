package cclf

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/CMSgov/bcda-app/bcda/cclf/metrics"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/optout"
)

type S3FileProcessor struct {
	Handler optout.S3FileHandler
}

func (processor *S3FileProcessor) LoadCclfFiles(path string) (cclfMap map[string]*cclfZipMetadata, skipped int, failed int, err error) {
	cclfMap = make(map[string]*cclfZipMetadata)
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

		for _, f := range zipReader.File {
			metadata, err := getCCLFFileMetadata(cmsID, f.Name)
			metadata.deliveryDate = *obj.LastModified

			if err != nil {
				// skipping files with a bad name.  An unknown file in this dir isn't a blocker
				processor.Handler.Warningf("Unknown file found: %s.", f.Name)
				continue
			}

			sub := cclfMap[metadata.acoID]
			if sub == nil {
				sub := &cclfZipMetadata{
					acoID:     metadata.acoID,
					zipReader: zipReader,
					zipCloser: zipCloser,
					filePath:  filepath.Join(bucket, *obj.Key),
				}
				cclfMap[metadata.acoID] = sub
			}

			if metadata.cclfNum == 0 {
				if sub.cclf0Metadata != nil {
					failed++
					processor.Handler.Errorf("Multiple CCLF0 files found in zip (%s/%s)", bucket, *obj.Key)
					delete(cclfMap, metadata.acoID)
					zipCloser()
					break
				}
				sub.cclf0Metadata = &metadata
				sub.cclf0File = f
			} else if metadata.cclfNum == 8 {
				if sub.cclf0Metadata != nil {
					failed++
					processor.Handler.Errorf("Multiple CCLF8 files found in zip (%s/%s)", bucket, *obj.Key)
					delete(cclfMap, metadata.acoID)
					zipCloser()
					break
				}
				sub.cclf8Metadata = &metadata
				sub.cclf8File = f
			} else {
				failed++
				processor.Handler.Errorf("Unexpected CCLF num %d processed (%s/%s)", metadata.cclfNum, bucket, *obj.Key)
				delete(cclfMap, metadata.acoID)
				zipCloser()
				break
			}
		}
	}

	return cclfMap, skipped, failed, err
}

func (processor *S3FileProcessor) CleanUpCCLF(ctx context.Context, cclfMap map[string]*cclfZipMetadata) error {
	errCount := 0

	for _, cclfZipMetadata := range cclfMap {
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

		processor.Handler.Infof("File %s successfully ingested and deleted from S3.\n", cclfZipMetadata.filePath)
	}

	if errCount > 0 {
		return fmt.Errorf("%d files could not be cleaned up", errCount)
	}

	return nil
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
