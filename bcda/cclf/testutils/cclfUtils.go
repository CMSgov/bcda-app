package testutils

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/cclf"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/utils"
)

// ImportCCLFPackage will copy the appropriate synthetic CCLF files, rename them,
// begin the import of those files and delete them from the place they were copied to after successful import.
func ImportCCLFPackage(acoSize, environment string, fileType models.CCLFFileType) (err error) {

	dir, err := ioutil.TempDir("", "*")
	if err != nil {
		return err
	}
	defer func() {
		err1 := os.RemoveAll(dir)
		if err == nil {
			err = err1
		}
	}()

	acoSize = strings.ToLower(acoSize)
	info, ok := map[string]struct {
		fileName string
		cmsID    string
	}{
		"dev":           {"dev", "A9994"},
		"dev-auth":      {"dev", "A9996"},
		"dev-cec":       {"dev", "E9994"},
		"dev-cec-auth":  {"dev", "E9996"},
		"dev-ng":        {"dev", "V994"},
		"dev-ng-auth":   {"dev", "V996"},
		"dev-ckcc":      {"dev", "C9994"},
		"dev-ckcc-auth": {"dev", "C9996"},
		"dev-kcf":       {"dev", "K9994"},
		"dev-kcf-auth":  {"dev", "K9996"},
		"dev-dc":        {"dev", "D9994"},
		"dev-dc-auth":   {"dev", "D9996"},
		"small":         {"small", "A9990"},
		"medium":        {"medium", "A9991"},
		"large":         {"large", "A9992"},
		"extra-large":   {"extra-large", "A9993"},
	}[acoSize]

	if !ok {
		return errors.New("invalid argument for ACO size")
	}

	switch environment {
	case
		"test",
		"test-new-beneficiaries":
	default:
		return errors.New("invalid argument for environment")
	}

	sourcedir := filepath.Join("shared_files/cclf/files/synthetic", environment, info.fileName)
	sourcedir, err = utils.GetDirPath(sourcedir)
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir(sourcedir)
	if err != nil {
		return err
	}

	var fileList []string
	var archiveName string

	now := time.Now()
	dateStr := fmt.Sprintf("%s.D%s.T%s0", now.Format("06"), now.Format("060102"), now.Format("150405"))
	suffix := fmt.Sprintf("%s%s", fileType, dateStr)
	for _, file := range files {
		var filename string
		archiveName = fmt.Sprintf("T.BCD.%s.ZC%s", info.cmsID, suffix)
		if strings.HasPrefix(info.cmsID, "A") {
			filename = fmt.Sprintf("T.BCD.%s.%s%s", info.cmsID, file.Name(), suffix)
		} else if strings.HasPrefix(info.cmsID, "E") {
			filename = fmt.Sprintf("T.CEC.%s%s", file.Name(), suffix)
		} else if hasAnyPrefix(info.cmsID, "V", "C", "K", "D") {
			filename = fmt.Sprintf("T.%s.ACO.%s%s", info.cmsID, file.Name(), suffix)
		}
		sourceFilename := fmt.Sprintf("%s/%s__%s", sourcedir, file.Name(), filename)
		fileList = append(fileList, sourceFilename)
	}

	newZipFile, err := os.Create(path.Join(dir, archiveName))
	if err != nil {
		return err
	}
	defer utils.CloseFileAndLogError(newZipFile)

	zipWriter := zip.NewWriter(newZipFile)

	// Add all 3 files to the same zip
	for _, f := range fileList {
		err = addFileToZip(zipWriter, f)
		if err != nil {
			return err
		}
	}

	_ = zipWriter.Close()
	success, failure, skipped, err := cclf.ImportCCLFDirectory(dir)
	if err != nil {
		return err
	}
	fmt.Printf("Completed CCLF import.  Successfully imported %d files.  Failed to import %d files.  Skipped %d files.  See logs for more details.\n", success, failure, skipped)
	if success != 2 {
		err = errors.New("did not import 2 files")
		return err
	}

	return
}

// addFileToZip adds the file to a zip archive. The filename supplied has the schema srcname__dstname
func addFileToZip(zipWriter *zip.Writer, filename string) error {
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

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
