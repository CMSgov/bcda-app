package testutils

import (
	"archive/zip"
	"errors"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/cclf"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
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

	var fileList []string
	var archiveName string

	dateStr := fmt.Sprintf("%s.D%s", time.Now().Format("06"), time.Now().Format("060102"))
	for _, file := range files {
		//timeStr := time.Now().Add(time.Minute * time.Duration(i-1)).Format("1504059")
		archiveName = fmt.Sprintf("T.BCD.A%s.ZCY%s.T%s", acoIDNum, dateStr, "0000000")
		filename := fmt.Sprintf("T.BCD.A%s.%sY%s.T%s", acoIDNum, file.Name(), dateStr, "0000000")
		sourceFilename := fmt.Sprintf("%s/%s__%s", sourcedir, file.Name(), filename)
		fileList = append(fileList, sourceFilename)
	}

	newZipFile, err := os.Create(fmt.Sprintf("%s/%s", DestDir, archiveName))
	if err != nil {
		return err
	}
	defer func() {
		if ferr := newZipFile.Close(); ferr != nil {
			err = ferr
		}
	}()

	zipWriter := zip.NewWriter(newZipFile)

	// Add all 3 files to the same zip
	for _, f := range fileList {
		err = AddFileToZip(zipWriter, f)
		if err != nil {
			return err
		}
	}

	_ = zipWriter.Close()
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

// AddFileToZip adds the file to a zip archive
func AddFileToZip(zipWriter *zip.Writer, filename string) (err error) {
	sourceData := strings.Split(filename, "__")
	src := sourceData[0]
	filename = sourceData[1]

	fileToZip, err := os.Open(filepath.Clean(src))
	if err != nil {
		return
	}
	defer func() {
		ferr := fileToZip.Close()
		if err == nil {
			err = ferr
		}
	}()

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return
	}

	// Using FileInfoHeader() above only uses the basename of the file. If we want
	// to preserve the folder structure we can overwrite this with the full path.
	header.Name = filename

	// Change to deflate to gain better compression
	// see http://golang.org/pkg/archive/zip/#pkg-constants
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return
	}
	_, err = io.Copy(writer, fileToZip)
	return
}
