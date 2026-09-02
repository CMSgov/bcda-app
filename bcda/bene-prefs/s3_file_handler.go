package beneprefs

import (
	"context"
	"fmt"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/models"
)

func LoadBenePrefsFiles(ctx context.Context, s3Helper bcdaaws.S3Helper, path string) (suppressList *[]*models.BenePrefsFilenameMetadata, skipped int, err error) {
	var result []*models.BenePrefsFilenameMetadata

	bucket, prefix := bcdaaws.ParseS3Uri(path)
	s3Objects, err := s3Helper.ListFiles(ctx, bucket, prefix)
	if err != nil {
		return &result, skipped, err
	}

	for _, obj := range s3Objects {
		metadata, err := parseMetadata(*obj.Key)
		metadata.FilePath = fmt.Sprintf("s3://%s/%s", bucket, *obj.Key)
		metadata.DeliveryDate = *obj.LastModified

		if err != nil {
			// Skip files with a bad name.  An unknown file in this dir isn't a blocker
			s3Helper.Logger.Errorf("Issue parsing filename into metadata: %v", err)
			skipped = skipped + 1
			continue
		}

		result = append(result, &metadata)
	}

	return &result, skipped, err
}

func CleanupBenePrefsFiles(ctx context.Context, s3Helper bcdaaws.S3Helper, suppresslist []*models.BenePrefsFilenameMetadata) error {
	errCount := 0

	for _, bpFile := range suppresslist {
		if !bpFile.Imported {
			// Don't do anything. The S3 bucket should have a retention policy that
			// automatically cleans up files after a specified period of time,
			s3Helper.Logger.Warningf("File %s was not imported successfully. Skipping cleanup", bpFile)
			continue
		}

		s3Helper.Logger.Infof("Cleaning up file %s\n", bpFile)
		err := s3Helper.Delete(ctx, bpFile.FilePath)

		if err != nil {
			errCount++
			continue
		}

		s3Helper.Logger.Infof("File %s successfully ingested and deleted from S3", bpFile)
	}

	if errCount > 0 {
		return fmt.Errorf("%d files could not be cleaned up", errCount)
	}

	return nil
}
