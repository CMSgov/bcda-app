package beneprefs

import (
	"bufio"
	"context"

	"github.com/CMSgov/bcda-app/bcda/models"
)

// File handlers can load opt out files from a given source and can optionally clean them up afterwards.
// This interface allows us to implement file loading from multiple sources, including local directories and AWS S3.
type BenePrefsFileHandler interface {
	// Load opt out files from the given path.
	//
	// Return a list of metadata parsed from valid filenames,
	// and the number of files skipped due to unknown filenames.
	LoadBenePrefsFiles(ctx context.Context, path string) (suppressList *[]*models.BenePrefsFilenameMetadata, skipped int, err error)
	// Cleanup any opt out files that were successfully imported, and handle
	// any files that failed to be imported.
	CleanupBenePrefsFiles(ctx context.Context, suppressList []*models.BenePrefsFilenameMetadata) error
	// Open a given opt out file, specified by the metadata struct.
	OpenFile(ctx context.Context, metadata *models.BenePrefsFilenameMetadata) (*bufio.Scanner, func(), error)
}
