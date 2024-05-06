package cclf

import (
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/log"
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
		// TODO: path or prefix
		optOut, _ := optout.IsOptOut(path)
		if optOut {
			handler.Infof("Skipping opt-out file: ", path)
			continue
		}

		zipReader, _, err := handler.OpenZipArchive(path)

		// validate the top level zipped folder
		cmsID, err := getCMSID(path)
		if err != nil {
			handler.Warningf("Skipping CCLF archive (%s): %s.", path, err)
			continue
		}

		supported := service.IsSupportedACO(cmsID)
		if !supported {
			handler.Errorf("Skipping CCLF archive (%s): cmsID %s not supported.", path, cmsID)
			continue
		}

		for _, f := range zipReader.File {
			metadata, err := getCCLFFileMetadata(cmsID, f.Name)
			metadata.filePath = path
			metadata.deliveryDate = *obj.LastModified

			if err != nil {
				// skipping files with a bad name.  An unknown file in this dir isn't a blocker
				msg := fmt.Sprintf("Unknown file found: %s.", f.Name)
				fmt.Println(msg)
				log.API.Error(msg)
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
