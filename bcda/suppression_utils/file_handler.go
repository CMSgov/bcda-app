package suppression_utils

import (
	"bufio"
)

type OptOutFileHandler interface {
	LoadSuppressionFiles() (suppressList []*SuppressionFileMetadata, skipped int, err error)
	CleanupSuppression(suppressList []*SuppressionFileMetadata) error
	OpenFile(metadata *SuppressionFileMetadata) (*bufio.Scanner, func(), error)
}
