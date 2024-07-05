package optout

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// LocalFileHandler manages files from local directories.
// This handler should only be used for local dev/testing now.
type LocalFileHandler struct {
	Logger                 logrus.FieldLogger
	PendingDeletionDir     string
	FileArchiveThresholdHr uint
}

func (handler *LocalFileHandler) LoadOptOutFiles(path string) (suppressList *[]*OptOutFilenameMetadata, skipped int, err error) {
	var result []*OptOutFilenameMetadata
	err = filepath.Walk(path, handler.getOptOutFileMetadata(&result, &skipped))
	return &result, skipped, err
}

func (handler *LocalFileHandler) getOptOutFileMetadata(suppresslist *[]*OptOutFilenameMetadata, skipped *int) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			var fileName = "nil"
			if info != nil {
				fileName = info.Name()
			}
			fmt.Printf("Error in checking suppression file %s: %s.\n", fileName, err)
			err = errors.Wrapf(err, "error in checking suppression file: %s,", fileName)
			handler.Logger.Error(err)
			return err
		}
		// Directories are not Suppression files
		if info.IsDir() {
			return nil
		}

		metadata, err := ParseMetadata(info.Name())
		metadata.FilePath = path
		metadata.DeliveryDate = info.ModTime()
		if err != nil {
			handler.Logger.Error(err)

			// skipping files with a bad name.  An unknown file in this dir isn't a blocker
			fmt.Printf("Unknown file found: %s.\n", metadata)
			handler.Logger.Errorf("Unknown file found: %s", metadata)
			*skipped = *skipped + 1

			deleteThreshold := time.Hour * time.Duration(handler.FileArchiveThresholdHr)
			if metadata.DeliveryDate.Add(deleteThreshold).Before(time.Now()) {
				newpath := fmt.Sprintf("%s/%s", handler.PendingDeletionDir, info.Name())
				err = os.Rename(metadata.FilePath, newpath)
				if err != nil {
					errmsg := fmt.Sprintf("error moving unknown file %s to pending deletion dir", metadata)
					err = errors.Wrap(err, errmsg)
					fmt.Println(errmsg)
					handler.Logger.Error(err)
					return err
				}
			}
			return nil
		}

		*suppresslist = append(*suppresslist, &metadata)
		return nil
	}
}

func (handler *LocalFileHandler) OpenFile(metadata *OptOutFilenameMetadata) (*bufio.Scanner, func(), error) {
	f, err := os.Open(metadata.FilePath)
	if err != nil {
		fmt.Printf("Could not read file %s.\n", metadata)
		err = errors.Wrapf(err, "could not read file %s", metadata)
		handler.Logger.Error(err)
		return nil, nil, err
	}

	sc := bufio.NewScanner(f)
	return sc, func() {
		if err := f.Close(); err != nil {
			handler.Logger.Error(err)
		}
	}, nil
}

func (handler *LocalFileHandler) CleanupOptOutFiles(suppresslist []*OptOutFilenameMetadata) error {
	errCount := 0
	for _, suppressionFile := range suppresslist {
		fmt.Printf("Cleaning up file %s.\n", suppressionFile)
		handler.Logger.Infof("Cleaning up file %s", suppressionFile)
		newpath := fmt.Sprintf("%s/%s", handler.PendingDeletionDir, suppressionFile.Name)
		if !suppressionFile.Imported {
			// check the timestamp on the failed files
			elapsed := time.Since(suppressionFile.DeliveryDate).Hours()

			if int(elapsed) > int(handler.FileArchiveThresholdHr) {
				err := os.Rename(suppressionFile.FilePath, newpath)
				if err != nil {
					errCount++
					errMsg := fmt.Sprintf("File %s failed to clean up properly: %v", suppressionFile, err)
					fmt.Println(errMsg)
					handler.Logger.Error(errMsg)
				} else {
					fmt.Printf("File %s never ingested, moved to the pending deletion dir.\n", suppressionFile)
					handler.Logger.Infof("File %s never ingested, moved to the pending deletion dir", suppressionFile)
				}
			}
		} else {
			// move the successful files to the deletion dir
			err := os.Rename(suppressionFile.FilePath, newpath)
			if err != nil {
				errCount++
				errMsg := fmt.Sprintf("File %s failed to clean up properly: %v", suppressionFile, err)
				fmt.Println(errMsg)
				handler.Logger.Error(errMsg)
			} else {
				fmt.Printf("File %s successfully ingested, moved to the pending deletion dir.\n", suppressionFile)
				handler.Logger.Infof("File %s successfully ingested, moved to the pending deletion dir", suppressionFile)
			}
		}
	}
	if errCount > 0 {
		return fmt.Errorf("%d files could not be cleaned up", errCount)
	}
	return nil
}
