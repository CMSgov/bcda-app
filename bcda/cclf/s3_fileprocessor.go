package cclf

import (
	"context"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/cclf/metrics"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/optout"
	"github.com/sirupsen/logrus"
)

type S3FileProcessor struct {
	Logger logrus.FieldLogger
	// Optional S3 endpoint to use for connection.
	Endpoint string
	// Optional role to assume when connecting to S3.
	AssumeRoleArn string
}

func (processor *S3FileProcessor) LoadCclfFiles(path string) (cclfMap map[string]map[metadataKey][]*cclfFileMetadata, skipped int, failed int, err error) {
	cclfMap = make(map[string]map[metadataKey][]*cclfFileMetadata)
	handler := optout.S3FileHandler{
		Logger:        processor.Logger,
		Endpoint:      processor.Endpoint,
		AssumeRoleArn: processor.AssumeRoleArn,
	}

	bucket, prefix := optout.ParseS3Uri(path)
	s3Objects, err := handler.ListFiles(bucket, prefix)

	if err != nil {
		return cclfMap, skipped, failed, err
	}

	for _, obj := range s3Objects {
		// ignore the opt out file, and don't add it to the skipped count
		optOut, _ := optout.IsOptOut(path)
		if optOut {
			handler.Infof("Skipping opt-out file: %s", path)
			continue
		}

		zipReader, _, err := handler.OpenZipArchive(path)

		if err != nil {
			failed++
			handler.Warningf("Failed to open CCLF archive (%s): %s.", path, err)
			continue
		}

		// validate the top level zipped folder
		cmsID, err := getCMSID(path)
		if err != nil {
			skipped++
			handler.Warningf("Skipping CCLF archive (%s): %s.", path, err)
			continue
		}

		supported := service.IsSupportedACO(cmsID)
		if !supported {
			skipped++
			handler.Errorf("Skipping CCLF archive (%s): cmsID %s not supported.", path, cmsID)
			continue
		}

		for _, f := range zipReader.File {
			metadata, err := getCCLFFileMetadata(cmsID, f.Name)
			metadata.filePath = path
			metadata.deliveryDate = *obj.LastModified

			if err != nil {
				// skipping files with a bad name.  An unknown file in this dir isn't a blocker
				skipped++
				handler.Errorf("Unknown file found: %s.", f.Name)
				continue
			}

			key := metadataKey{perfYear: metadata.perfYear, fileType: metadata.fileType}
			sub := cclfMap[metadata.acoID]
			if sub == nil {
				sub = make(map[metadataKey][]*cclfFileMetadata)
				cclfMap[metadata.acoID] = sub
			}
			sub[key] = append(sub[key], &metadata)
		}
	}

	return cclfMap, skipped, failed, err
}

func (processor *S3FileProcessor) CleanUpCCLF(ctx context.Context, cclfMap map[string]map[metadataKey][]*cclfFileMetadata) error {
	handler := optout.S3FileHandler{
		Logger:        processor.Logger,
		Endpoint:      processor.Endpoint,
		AssumeRoleArn: processor.AssumeRoleArn,
	}

	errCount := 0

	for _, cclfFileMap := range cclfMap {
		for _, cclfFileList := range cclfFileMap {
			for _, cclf := range cclfFileList {
				close := metrics.NewChild(ctx, fmt.Sprintf("cleanUpCCLF%d", cclf.cclfNum))
				defer close()

				if !cclf.imported {
					// Don't do anything. The S3 bucket should have a retention policy that
					// automatically cleans up files after a specified period of time,
					handler.Warningf("File %s was not imported successfully. Skipping cleanup.\n", cclf.filePath)
					continue
				}

				handler.Infof("Cleaning up file %s\n", cclf.filePath)
				err := handler.Delete(cclf.filePath)

				if err != nil {
					errCount++
					continue
				}

				handler.Infof("File %s successfully ingested and deleted from S3.\n", cclf.filePath)
			}
		}
	}

	if errCount > 0 {
		return fmt.Errorf("%d files could not be cleaned up", errCount)
	}

	return nil
}
