package optout

import (
	"bufio"
)

type OptOutFileHandler interface {
	LoadOptOutFiles(path string) (suppressList *[]*OptOutFilenameMetadata, skipped int, err error)
	CleanupOptOutFiles(suppressList []*OptOutFilenameMetadata) error
	OpenFile(metadata *OptOutFilenameMetadata) (*bufio.Scanner, func(), error)
}
