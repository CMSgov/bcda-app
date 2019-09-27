package testutils

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/cclf"
	"github.com/CMSgov/bcda-app/bcda/utils"
)

const DestDir = "tempCCLFDir/"

// ImportCCLFPackage will copy the appropriate synthetic CCLF files, rename them,
// begin the import of those files and delete them from the place they were copied to after successful import.

func ImportCCLFPackage(acoSize, environment string) (err error) {
	acoSize = strings.ToLower(acoSize)
	acoIDNum := map[string]string{
		"dev":         "9994",
		"dev-auth":    "9996",
		"small":       "9990",
		"medium":      "9991",
		"large":       "9992",
		"extra-large": "9993",
	}[acoSize]
	if acoIDNum == "" {
		return errors.New("invalid argument for ACO size")
	}

	switch environment {
	case
		"test",
		"unit-test":
	default:
		return errors.New("invalid argument for environment")
	}

	sourcedir := filepath.Join("shared_files/cclf/files/synthetic", environment, acoSize)
	sourcedir, err = utils.GetDirPath(sourcedir)
	if err != nil {
		return err
	}

	if _, err := os.Stat(DestDir); os.IsNotExist(err) {
		err = os.Mkdir(DestDir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	files, err := ioutil.ReadDir(sourcedir)
	if err != nil {
		return err
	}

	dateStr := fmt.Sprintf("%s.D%s", time.Now().Format("06"), time.Now().Format("060102"))
	for _, file := range files {
		filename := fmt.Sprintf("T.A%s.ACO.%sY%s.T%s", acoIDNum, file.Name(), dateStr, time.Now().Format("1504059"))
		archiveName := fmt.Sprintf("T.BCD.ACO.%sY%s.T%s%s", file.Name(), dateStr, acoIDNum, "000")
		err = zipTo(fmt.Sprintf("%s/%s", sourcedir, file.Name()), filename, fmt.Sprintf("%s/%s", DestDir, archiveName))
		if err != nil {
			return err
		}
	}

	success, failure, skipped, err := cclf.ImportCCLFDirectory(DestDir)
	if err != nil {
		return err
	}
	fmt.Printf("Completed CCLF import.  Successfully imported %d files.  Failed to import %d files.  Skipped %d files.  See logs for more details.\n", success, failure, skipped)
	if success == 2 {
		_, err = utils.DeleteDirectoryContents(DestDir)
		return err
	} else {
		err = errors.New("did not import 2 files")
		return err
	}
}

func zipTo(src, dstFile, dstArchive string) error {
	srcFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !srcFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}
	defer source.Close()

	newZipFile, err := os.Create(dstArchive)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	header, err := zip.FileInfoHeader(srcFileStat)
	if err != nil {
		return err
	}

	header.Name = dstFile
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, source)
	return err
}
