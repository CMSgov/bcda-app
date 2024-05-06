package cclf

import (
	"archive/zip"
)

// File handlers can load opt out files from a given source and can optionally clean them up afterwards.
// This interface allows us to implement file loading from multiple sources, including local directories and AWS S3.
type CclfFileHandler interface {
	ProcessCCLFArchives(rootPath string) (map[string]map[metadataKey][]*cclfFileMetadata, int, int, error)
	OpenZipArchive(name string) (*zip.ReadCloser, error)
}
