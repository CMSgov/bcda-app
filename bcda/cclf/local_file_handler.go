package cclf

import (
	"archive/zip"
	"path/filepath"
)

type LocalFileHandler struct {
}

func (handler *LocalFileHandler) OpenZipArchive(path string) (*zip.ReadCloser, func() error, error) {
	reader, err := zip.OpenReader(filepath.Clean(path))
	if err != nil {
		return reader, func() error { return nil }, err
	}

	return reader, reader.Close, err
}

func (handler *LocalFileHandler) ProcessCCLFArchives(rootPath string) (map[string]map[metadataKey][]*cclfFileMetadata, int, int, error) {
	return processCCLFArchives(rootPath)
}
