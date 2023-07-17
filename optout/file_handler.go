package optout

import (
	"bufio"
)

type OptOutFileHandler interface {
	LoadSuppressionFiles(path string) (suppressList []*SuppressionFileMetadata, skipped int, err error)
	CleanupSuppression(suppressList []*SuppressionFileMetadata) error
	OpenFile(metadata *SuppressionFileMetadata) (*bufio.Scanner, func(), error)
}
