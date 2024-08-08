package cclf

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CMSgov/bcda-app/bcda/cclf/metrics"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/CMSgov/bcda-app/optout"

	"github.com/pkg/errors"
)

type metadataKey struct {
	perfYear int
	fileType models.CCLFFileType
}

type LocalFileProcessor struct {
	Handler optout.LocalFileHandler
}

func (processor *LocalFileProcessor) LoadCclfFiles(path string) (cclfList map[string]*cclfZipMetadata, skipped int, failed int, err error) {
	return processCCLFArchives(path)
}

// processCCLFArchives walks through all of the CCLF files captured in the root path and generates
// a mapping between CMS_ID + perf year and associated CCLF Metadata
func processCCLFArchives(rootPath string) (map[string]*cclfZipMetadata, int, int, error) {
	p := &processor{0, 0, make(map[string]*cclfZipMetadata)}
	if err := filepath.Walk(rootPath, p.walk); err != nil {
		return nil, 0, 0, err
	}
	return p.cclfMap, p.skipped, p.failure, nil
}

type processor struct {
	skipped int
	failure int
	cclfMap map[string]*cclfZipMetadata
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
		err := zipReader.Close()
		if err != nil {
			log.API.Warningf("Could not close zip archive %s", path)
		}
	}

	if err != nil {
		modTime := info.ModTime()
		if stillDownloading(modTime) {
			// ignore downlading file, and don't add it to the skipped count
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

		sub := p.cclfMap[metadata.acoID]
		if sub == nil {
			sub := &cclfZipMetadata{
				acoID:     metadata.acoID,
				zipReader: &zipReader.Reader,
				zipCloser: zipCloser,
				filePath:  path,
			}
			p.cclfMap[metadata.acoID] = sub
		}

		if metadata.cclfNum == 0 {
			if sub.cclf0Metadata != nil {
				p.failure++
				log.API.Errorf("Multiple CCLF0 files found in zip (%s)", path)
				delete(p.cclfMap, metadata.acoID)
				zipCloser()
				break
			}
			sub.cclf0Metadata = &metadata
			sub.cclf0File = f
		} else if metadata.cclfNum == 8 {
			if sub.cclf0Metadata != nil {
				p.failure++
				log.API.Errorf("Multiple CCLF8 files found in zip (%s)", path)
				delete(p.cclfMap, metadata.acoID)
				zipCloser()
				break
			}
			sub.cclf8Metadata = &metadata
			sub.cclf8File = f
		} else {
			p.failure++
			log.API.Errorf("Unexpected CCLF num %d processed (%s)", metadata.cclfNum, path)
			delete(p.cclfMap, metadata.acoID)
			zipCloser()
			break
		}
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

func (processor *LocalFileProcessor) CleanUpCCLF(ctx context.Context, cclfMap map[string]*cclfZipMetadata) error {
	errCount := 0
	for _, cclfZipMetadata := range cclfMap {
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
						processor.Handler.Logger.Error("File %s failed to clean up properly: %v", cclfZipMetadata.filePath, err)
					} else {
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
					processor.Handler.Logger.Error("File %s failed to clean up properly: %v", cclfZipMetadata.filePath, err)
				} else {
					processor.Handler.Logger.Infof("File %s successfully ingested, moved to the pending deletion dir", cclfZipMetadata.filePath)
				}
			}
		}()
	}
	if errCount > 0 {
		return fmt.Errorf("%d files could not be cleaned up", errCount)
	}
	return nil
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
