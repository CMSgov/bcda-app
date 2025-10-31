package optout

import (
	"bufio"
	"context"
)

// File handlers can load opt out files from a given source and can optionally clean them up afterwards.
// This interface allows us to implement file loading from multiple sources, including local directories and AWS S3.
type OptOutFileHandler interface {
	// Load opt out files from the given path.
	//
	// Return a list of metadata parsed from valid filenames,
	// and the number of files skipped due to unknown filenames.
	LoadOptOutFiles(ctx context.Context, path string) (suppressList *[]*OptOutFilenameMetadata, skipped int, err error)
	// Cleanup any opt out files that were successfully imported, and handle
	// any files that failed to be imported.
	CleanupOptOutFiles(ctx context.Context, suppressList []*OptOutFilenameMetadata) error
	// Open a given opt out file, specified by the metadata struct.
	OpenFile(ctx context.Context, metadata *OptOutFilenameMetadata) (*bufio.Scanner, func(), error)
}
