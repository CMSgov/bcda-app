package cclf

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CMSgov/bcda-app/bcda/cclf/metrics"
	ers "github.com/CMSgov/bcda-app/bcda/errors"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/CMSgov/bcda-app/optout"
	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
)

type LocalFileProcessor struct {
	Handler optout.LocalFileHandler
}

func (processor *LocalFileProcessor) LoadCclfFiles(path string) (cclfList map[string][]*cclfZipMetadata, skipped int, failed int, err error) {
	return processCCLFArchives(path)
}

// processCCLFArchives walks through all of the CCLF files captured in the root path and generates
// a mapping between CMS_ID + perf year and associated CCLF Metadata
func processCCLFArchives(rootPath string) (map[string][]*cclfZipMetadata, int, int, error) {
	p := &processor{0, 0, make(map[string][]*cclfZipMetadata)}
	if err := filepath.Walk(rootPath, p.walk); err != nil {
		return nil, 0, 0, err
	}
	return p.cclfMap, p.skipped, p.failure, nil
}

type processor struct {
	skipped int
	failure int
	cclfMap map[string][]*cclfZipMetadata
}

func (p *processor) walk(path string, info os.FileInfo, err error) error {
	if err != nil {
		// In case the caller supplied an err, we know that info is nil
		// See: https://golang.org/pkg/path/filepath/#WalkFunc
		var fileName = "nil"
		err = errors.Wrapf(err, "error in sorting cclf file: %v,", fileName)
		fmt.Println(err.Error())
		log.API.Error(err)
		return err
	}

	if info.IsDir() {
		msg := fmt.Sprintf("Unable to sort %s: directory, not a CCLF archive.", path)
		fmt.Println(msg)
		log.API.Warn(msg)
		return nil
	}

	// ignore the opt out file, and don't add it to the skipped count
	optOut, _ := optout.IsOptOut(info.Name())
	if optOut {
		fmt.Print("Skipping opt-out file: ", info.Name())
		log.API.Info("Skipping opt-out file: ", info.Name())
		return nil
	}

	zipFile := filepath.Clean(path)
	zipReader, err := zip.OpenReader(zipFile)
	zipCloser := func() {
		if zipReader != nil {
			err := zipReader.Close()
			if err != nil {
				log.API.Warningf("Could not close zip archive %s", path)
			}
		}
	}

	if err != nil {
		modTime := info.ModTime()
		if stillDownloading(modTime) {
			// ignore downloading file, and don't add it to the skipped count
			msg := fmt.Sprintf("Skipping %s: file was last modified on: %s and is still downloading. err: %s", path, modTime, err.Error())
			fmt.Println(msg)
			log.API.Warn(msg)
			zipCloser()
			return nil
		}

		p.failure = p.failure + 1
		msg := fmt.Errorf("Corrupted %s: file could not be opened as a CCLF archive. %s", path, err.Error())
		fmt.Println(msg)
		log.API.Error(msg)
		zipCloser()
		return nil
	}

	// validate the top level zipped folder
	cmsID, err := getCMSID(info.Name())
	if err != nil {
		zipCloser()
		return p.handleArchiveError(path, info, err)
	}

	supported := service.IsSupportedACO(cmsID)
	if !supported {
		zipCloser()
		return p.handleArchiveError(path, info, fmt.Errorf("cmsID %s not supported", cmsID))
	}

	var cclf0Metadata, cclf8Metadata *cclfFileMetadata
	var cclf0File, cclf8File *zip.File
	var readError error

	for _, f := range zipReader.File {
		metadata, err := getCCLFFileMetadata(cmsID, f.Name)
		metadata.deliveryDate = info.ModTime()

		if err != nil {
			// skipping files with a bad name.  An unknown file in this dir isn't a blocker
			msg := fmt.Sprintf("Unknown file found: %s.", f.Name)
			fmt.Println(msg)
			log.API.Error(msg)
			continue
		}

		if metadata.cclfNum == 0 {
			if cclf0Metadata != nil {
				readError = fmt.Errorf("Multiple CCLF0 files found in zip (%s)", path)
				break
			}
			cclf0Metadata = &metadata
			cclf0File = f
		} else if metadata.cclfNum == 8 {
			if cclf8Metadata != nil {
				readError = fmt.Errorf("Multiple CCLF8 files found in zip (%s)", path)
				break
			}
			cclf8Metadata = &metadata
			cclf8File = f
		} else {
			readError = fmt.Errorf("Unexpected CCLF num %d processed (%s)", metadata.cclfNum, path)
			break
		}
	}

	if readError != nil {
		p.failure++
		println(readError.Error())
		log.API.Errorf(readError.Error())
		zipCloser()
	} else if cclf0Metadata == nil || cclf8Metadata == nil {
		p.failure++
		fmt.Printf("Missing CCLF0 or CCLF8 file in zip (%s)\n", path)
		log.API.WithFields(logrus.Fields{
			"missingCCLF0": cclf0Metadata == nil,
			"missingCCLF8": cclf8Metadata == nil,
		}).Errorf("Missing CCLF0 or CCLF8 file in zip (%s)", path)
		zipCloser()
	} else {
		zipMetadata := cclfZipMetadata{
			acoID:         cmsID,
			zipReader:     &zipReader.Reader,
			zipCloser:     zipCloser,
			cclf0Metadata: *cclf0Metadata,
			cclf8Metadata: *cclf8Metadata,
			cclf0File:     *cclf0File,
			cclf8File:     *cclf8File,
			filePath:      path,
		}

		p.cclfMap[cmsID] = append(p.cclfMap[cmsID], &zipMetadata)
	}

	return nil
}

func (p *processor) handleArchiveError(path string, info os.FileInfo, cause error) error {
	msg := fmt.Sprintf("Skipping CCLF archive (%s): %s.", info.Name(), cause)
	fmt.Println(msg)
	log.API.Warn(msg)
	err := checkDeliveryDate(path, info.ModTime())
	if err != nil {
		err = fmt.Errorf("error moving unknown file %s to pending deletion dir", path)
		fmt.Println(err.Error())
		log.API.Error(err)
	}

	return err
}

func checkDeliveryDate(folderPath string, deliveryDate time.Time) error {
	deleteThreshold := time.Hour * time.Duration(utils.GetEnvInt("BCDA_ETL_FILE_ARCHIVE_THRESHOLD_HR", 72))
	if deliveryDate.Add(deleteThreshold).Before(time.Now()) {
		folderName := filepath.Base(folderPath)
		newpath := fmt.Sprintf("%s/%s", conf.GetEnv("PENDING_DELETION_DIR"), folderName)
		err := os.Rename(folderPath, newpath)
		if err != nil {
			return err
		}
	}
	return nil
}

func stillDownloading(modTime time.Time) bool {
	// modified date < 1 min: still downloading
	now := time.Now()
	oneMinuteAgo := now.Add(time.Duration(-1) * time.Minute)

	return modTime.After(oneMinuteAgo)
}

func (processor *LocalFileProcessor) CleanUpCCLF(ctx context.Context, cclfMap map[string][]*cclfZipMetadata) (deletedCount int, err error) {
	errCount := 0
	for acoID := range cclfMap {
		for _, cclfZipMetadata := range cclfMap[acoID] {
			func() {
				close := metrics.NewChild(ctx, "cleanUpCCLFZip")
				defer close()

				processor.Handler.Logger.Infof("Cleaning up file %s.\n", cclfZipMetadata.filePath)
				folderName := filepath.Base(cclfZipMetadata.filePath)
				newpath := fmt.Sprintf("%s/%s", conf.GetEnv("PENDING_DELETION_DIR"), folderName)
				if !cclfZipMetadata.imported {
					// check the timestamp on the failed files
					elapsed := time.Since(cclfZipMetadata.cclf0Metadata.deliveryDate).Hours()
					deleteThreshold := utils.GetEnvInt("BCDA_ETL_FILE_ARCHIVE_THRESHOLD_HR", 72)
					if int(elapsed) > deleteThreshold {
						if _, err := os.Stat(newpath); err == nil {
							return
						}
						// move the (un)successful files to the deletion dir
						err := os.Rename(cclfZipMetadata.filePath, newpath)
						if err != nil {
							errCount++
							processor.Handler.Logger.Errorf("File %s failed to clean up properly: %v", cclfZipMetadata.filePath, err)
						} else {
							deletedCount++
							processor.Handler.Logger.Infof("File %s never ingested, moved to the pending deletion dir", cclfZipMetadata.filePath)
						}
					}
				} else {
					if _, err := os.Stat(newpath); err == nil {
						return
					}
					// move the successful files to the deletion dir
					err := os.Rename(cclfZipMetadata.filePath, newpath)
					if err != nil {
						errCount++
						processor.Handler.Logger.Errorf("File %s failed to clean up properly: %v", cclfZipMetadata.filePath, err)
					} else {
						deletedCount++
						processor.Handler.Logger.Infof("File %s successfully ingested, moved to the pending deletion dir", cclfZipMetadata.filePath)
					}
				}
			}()
		}
	}

	if errCount > 0 {
		return deletedCount, fmt.Errorf("%d files could not be cleaned up", errCount)
	}
	return deletedCount, nil
}

func (processor *LocalFileProcessor) OpenZipArchive(filePath string) (*zip.Reader, func(), error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, nil, err
	}

	return &reader.Reader, func() {
		err := reader.Close()
		if err != nil {
			processor.Handler.Logger.Warningf("Could not close zip archive %s", filePath)
		}
	}, err
}

func (processor *LocalFileProcessor) CleanUpCSV(file csvFile) error {
	var err error

	func() {
		close := metrics.NewChild(context.Background(), "cleanUpCSV")
		defer close()

		processor.Handler.Logger.Infof("Cleaning up file %s.\n", file.metadata.name)
		folderName := filepath.Base(file.filepath)
		newpath := fmt.Sprintf("%s/%s", conf.GetEnv("PENDING_DELETION_DIR"), folderName)
		if !file.imported {
			// check the timestamp on the failed files
			elapsed := time.Since(file.metadata.deliveryDate).Hours()
			deleteThreshold := utils.GetEnvInt("BCDA_ETL_FILE_ARCHIVE_THRESHOLD_HR", 72)
			if int(elapsed) > deleteThreshold {
				if _, err = os.Stat(newpath); err == nil {
					return
				}
				// move the (un)successful files to the deletion dir
				err = os.Rename(file.filepath, newpath)
				if err != nil {
					processor.Handler.Logger.Errorf("File %s failed to clean up properly: %v", file.filepath, err)
				} else {
					processor.Handler.Logger.Infof("File %s never ingested, moved to the pending deletion dir", file.filepath)
				}
			}
		} else {
			if _, err = os.Stat(newpath); err == nil {
				return
			}
			err = os.Rename(file.filepath, newpath)
			if err != nil {
				processor.Handler.Logger.Errorf("File %s failed to clean up properly: %v", file.filepath, err)

			} else {
				processor.Handler.Logger.Infof("File %s successfully ingested, moved to the pending deletion dir", file.filepath)
			}
		}
	}()
	return err
}

func (processor *LocalFileProcessor) LoadCSV(filepath string) (*bytes.Reader, func(), error) {
	optOut, _ := optout.IsOptOut(filepath)
	if optOut {
		return nil, nil, &ers.IsOptOutFile{}
	}
	byte_arr, err := os.ReadFile(filepath)
	if err != nil {
		return nil, nil, err
	}
	reader := bytes.NewReader(byte_arr)

	return reader, func() {}, err

}
