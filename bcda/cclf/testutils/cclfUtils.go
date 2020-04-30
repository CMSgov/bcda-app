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
		"test-new-beneficiaries",
		"unit-test",
		"unit-test-new-beneficiaries":
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

	now := time.Now()
	dateStr := fmt.Sprintf("Y%s.D%s.T%s0", now.Format("06"), now.Format("060102"), now.Format("150405"))
	for _, file := range files {
		archiveName = fmt.Sprintf("T.BCD.A%s.ZC%s", acoIDNum, dateStr)
		filename := fmt.Sprintf("T.BCD.A%s.%s%s", acoIDNum, file.Name(), dateStr)
		sourceFilename := fmt.Sprintf("%s/%s__%s", sourcedir, file.Name(), filename)
		fileList = append(fileList, sourceFilename)
	}

	newZipFile, err := os.Create(fmt.Sprintf("%s/%s", DestDir, archiveName))
	if err != nil {
		return err
	}
	defer utils.CloseFileAndLogError(newZipFile)

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
func AddFileToZip(zipWriter *zip.Writer, filename string) error {
	sourceData := strings.Split(filename, "__")
	src := sourceData[0]
	filename = sourceData[1]

	fileToZip, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}
	defer utils.CloseFileAndLogError(fileToZip)

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// Using FileInfoHeader() above only uses the basename of the file. If we want
	// to preserve the folder structure we can overwrite this with the full path.
	header.Name = filename

	// Change to deflate to gain better compression
	// see http://golang.org/pkg/archive/zip/#pkg-constants
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}
