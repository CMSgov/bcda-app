package optout

import (
	"bufio"
)

// File handlers can load opt out files from a given source and can optionally clean them up afterwards.
// This interface allows us to implement file loading from multiple sources, including local directories and AWS S3.
type OptOutFileHandler interface {
	LoadOptOutFiles(path string) (suppressList *[]*OptOutFilenameMetadata, skipped int, err error)
	CleanupOptOutFiles(suppressList []*OptOutFilenameMetadata) error
	OpenFile(metadata *OptOutFilenameMetadata) (*bufio.Scanner, func(), error)
}
