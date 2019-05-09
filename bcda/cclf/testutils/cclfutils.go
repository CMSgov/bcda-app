package testutils

import (
	"errors"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/cclf"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// ImportCCLFPackage will copy the appropriate synthetic CCLF files, rename them,
// begin the import of those files and delete them from the place they were copied to after successful import.

func ImportCCLFPackage(acoSize, environment string) (err error) {
	acoSize = strings.ToLower(acoSize)
	// validation is here because this will get called from other locations than the CLI.
	switch acoSize {
	case
		"dev",
		"small",
		"medium",
		"large":

	default:
		return errors.New("invalid argument for aco size")
	}

	if environment != "test" {
		return errors.New("invalid argument for environment")
	}
	sourcedir := fmt.Sprintf("../shared_files/syntheticCCLFFiles/%s/%s", environment, acoSize)
	destdir := "tempCCLFDir/"
	if _, err := os.Stat(destdir); os.IsNotExist(err) {
		os.Mkdir(destdir, os.ModePerm)
	}

	dateString := fmt.Sprintf("%s.D%s.T%s", time.Now().Format("06"), time.Now().Format("060102"), time.Now().Format("1504059"))

	files, err := ioutil.ReadDir(sourcedir)
	if err != nil {
		return err
	}
	for _, file := range files {
		err = copyFiles(fmt.Sprintf("%s/%s", sourcedir, file.Name()), fmt.Sprintf("%s/%s%s", destdir, file.Name(), dateString))
	}

	success, failure, skipped, err := cclf.ImportCCLFDirectory(destdir)
	fmt.Println("Completed CCLF import.  Successfully imported %v files.  Failed to import %v files.  Skipped %v files.  See logs for more details.", success, failure, skipped)
	if success == 3 {
		_, err = cclf.DeleteDirectoryContents(destdir)
		return err
	} else {
		err = errors.New("did not import 3 files")
		return err
	}
}

func copyFiles(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}
