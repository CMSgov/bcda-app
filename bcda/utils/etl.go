package utils

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

func DeleteDirectoryContents(dirToDelete string) (filesDeleted int, err error) {
	fmt.Printf("Preparing to delete directory %v.\n", dirToDelete)
	log.Infof("Preparing to delete directory %v", dirToDelete)
	f, err := os.Open(filepath.Clean(dirToDelete))
	if err != nil {
		fmt.Printf("Could not open dir: %s.\n", dirToDelete)
		err = errors.Wrapf(err, "could not open dir: %s", dirToDelete)
		log.Error(err)
		return 0, err
	}
	files, err := f.Readdir(-1)
	if err != nil {
		fmt.Printf("Error reading files from dir: %s.\n", f.Name())
		err = errors.Wrapf(err, "error reading files from dir: %s", f.Name())
		log.Error(err)
		return 0, err
	}
	err = f.Close()
	if err != nil {
		fmt.Printf("Error closing dir: %s.\n", f.Name())
		err = errors.Wrapf(err, "error closing dir: %s", f.Name())
		log.Error(err)
		return 0, err
	}

	for _, file := range files {
		fmt.Printf("Deleting %s.\n", file.Name())
		log.Infof("deleting %s ", file.Name())
		err = os.Remove(filepath.Join(dirToDelete, file.Name()))
		if err != nil {
			fmt.Printf("Error deleting file: %s from dir: %s.\n", file.Name(), dirToDelete)
			err = errors.Wrapf(err, "error deleting file: %s from dir: %s", file.Name(), dirToDelete)
			log.Error(err)
			return 0, err
		}
	}

	fmt.Printf("Successfully deleted all files from dir: %s.\n", dirToDelete)
	log.Infof("Successfully deleted all files from dir: %s", dirToDelete)
	return len(files), nil
}
