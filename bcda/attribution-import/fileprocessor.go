package attributionimport

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/service"
)

func LoadCclfFiles(ctx context.Context, fileHelper bcdaaws.S3Helper, path string) (cclfMap map[string][]*cclfZipMetadata, skipped int, failed int, err error) {
	cclfMap = make(map[string][]*cclfZipMetadata)
	bucket, prefix := bcdaaws.ParseS3Uri(path)
	s3Objects, err := fileHelper.ListFiles(ctx, bucket, prefix)

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
			fileHelper.Logger.Errorf("Skipping CCLF archive (%s/%s): %v", bucket, *obj.Key, err)
			continue
		}

		supported := cfg.IsSupportedACO(cmsID)
		if !supported {
			fileHelper.Logger.Errorf("Skipping CCLF archive (%s/%s): cmsID %s not supported.", bucket, *obj.Key, cmsID)
			continue
		}

		zipReader, zipCloser, err := OpenZipArchive(ctx, fileHelper, filepath.Join(bucket, *obj.Key))

		if err != nil {
			failed++
			fileHelper.Logger.Errorf("Failed to open CCLF archive (%s/%s): %s.", bucket, *obj.Key, err)
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
				fileHelper.Logger.Errorf("Issue parsing filename into metadata: %v", err)
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
			fileHelper.Logger.Errorf(readError.Error())
			zipCloser()
		} else if cclf0Metadata == nil || cclf8Metadata == nil {
			failed++
			fileHelper.Logger.Errorf("Missing CCLF0 or CCLF8 file in zip (%s/%s)", bucket, *obj.Key)
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

func CleanUpCCLF(ctx context.Context, fileHelper bcdaaws.S3Helper, cclfMap map[string][]*cclfZipMetadata) (deletedCount int, err error) {
	errCount := 0

	for acoID := range cclfMap {
		for _, cclfZipMetadata := range cclfMap[acoID] {
			if !cclfZipMetadata.imported {
				// Don't do anything. The S3 bucket should have a retention policy that
				// automatically cleans up files after a specified period of time.
				fileHelper.Logger.Warningf("File %s was not imported successfully. Skipping cleanup.", cclfZipMetadata.filePath)
				continue
			}

			fileHelper.Logger.Infof("Cleaning up file %s", cclfZipMetadata.filePath)
			err := fileHelper.Delete(ctx, cclfZipMetadata.filePath)

			if err != nil {
				errCount++
				continue
			}

			deletedCount++
			fileHelper.Logger.Infof("File %s successfully ingested and deleted from S3.", cclfZipMetadata.filePath)
		}
	}

	if errCount > 0 {
		return deletedCount, fmt.Errorf("%d files could not be cleaned up", errCount)
	}

	return deletedCount, nil
}

func OpenZipArchive(ctx context.Context, fileHelper bcdaaws.S3Helper, filePath string) (*zip.Reader, func(), error) {
	byte_arr, err := fileHelper.OpenFileAsBytes(ctx, filePath)

	if err != nil {
		fileHelper.Logger.Errorf("Failed to download %s", filePath)
		return nil, nil, err
	}

	reader, err := zip.NewReader(bytes.NewReader(byte_arr), int64(len(byte_arr)))
	return reader, func() {}, err
}

func CleanUpCSV(ctx context.Context, fileHelper bcdaaws.S3Helper, file csvFile) error {
	if !file.imported {
		// Don't do anything. The S3 bucket should have a retention policy that
		// automatically cleans up files after a specified period of time.
		fileHelper.Logger.Warningf("File %s was not imported successfully. Skipping cleanup.", file.filepath)
		return nil
	}

	fileHelper.Logger.Infof("Cleaning up file %s", file.filepath)
	err := fileHelper.Delete(ctx, file.filepath)

	if err != nil {
		fileHelper.Logger.Errorf("Failed to clean up file %s: %v", file.filepath, err)
		return err
	}

	fileHelper.Logger.Infof("File %s successfully ingested and deleted from S3.", file.filepath)
	return nil
}

func LoadCSV(ctx context.Context, fileHelper bcdaaws.S3Helper, filepath string) (*bytes.Reader, func(), error) {
	byte_arr, err := fileHelper.OpenFileAsBytes(ctx, filepath)
	if err != nil {
		fileHelper.Logger.Errorf("Failed to download %s", filepath)
		return nil, nil, err
	}

	reader := bytes.NewReader(byte_arr)
	return reader, func() {}, err
}
