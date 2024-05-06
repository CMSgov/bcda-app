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

type LocalFileProcessor struct{}

func (processor *LocalFileProcessor) LoadCclfFiles(path string) (cclfList map[string]map[metadataKey][]*cclfFileMetadata, skipped int, failed int, err error) {
	return processCCLFArchives(path)
}

// processCCLFArchives walks through all of the CCLF files captured in the root path and generates
// a mapping between CMS_ID + perf year and associated CCLF Metadata
func processCCLFArchives(rootPath string) (map[string]map[metadataKey][]*cclfFileMetadata, int, int, error) {
	p := &processor{0, 0, make(map[string]map[metadataKey][]*cclfFileMetadata)}
	if err := filepath.Walk(rootPath, p.walk); err != nil {
		return nil, 0, 0, err
	}
	return p.cclfMap, p.skipped, p.failure, nil
}

type processor struct {
	skipped int
	failure int
	cclfMap map[string]map[metadataKey][]*cclfFileMetadata
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

	if err != nil {
		modTime := info.ModTime()
		if stillDownloading(modTime) {
			// ignore downlading file, and don't add it to the skipped count
			msg := fmt.Sprintf("Skipping %s: file was last modified on: %s and is still downloading. err: %s", path, modTime, err.Error())
			fmt.Println(msg)
			log.API.Warn(msg)
			return nil
		}

		p.failure = p.failure + 1
		msg := fmt.Errorf("Corrupted %s: file could not be opened as a CCLF archive. %s", path, err.Error())
		fmt.Println(msg)
		log.API.Error(msg)
		return nil
	}

	if err = zipReader.Close(); err != nil {
		fmt.Printf("Failed to close zip file %s\n", err.Error())
		log.API.Warnf("Failed to close zip file %s", err.Error())
	}

	// validate the top level zipped folder
	cmsID, err := getCMSID(info.Name())
	if err != nil {
		return p.handleArchiveError(path, info, err)
	}

	supported := service.IsSupportedACO(cmsID)
	if !supported {
		return p.handleArchiveError(path, info, fmt.Errorf("cmsID %s not supported", cmsID))
	}

	for _, f := range zipReader.File {
		metadata, err := getCCLFFileMetadata(cmsID, f.Name)
		metadata.filePath = path
		metadata.deliveryDate = info.ModTime()

		if err != nil {
			// skipping files with a bad name.  An unknown file in this dir isn't a blocker
			msg := fmt.Sprintf("Unknown file found: %s.", f.Name)
			fmt.Println(msg)
			log.API.Error(msg)
			continue
		}

		key := metadataKey{perfYear: metadata.perfYear, fileType: metadata.fileType}
		sub := p.cclfMap[metadata.acoID]
		if sub == nil {
			sub = make(map[metadataKey][]*cclfFileMetadata)
			p.cclfMap[metadata.acoID] = sub
		}
		sub[key] = append(sub[key], &metadata)
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

func (processor *LocalFileProcessor) CleanUpCCLF(ctx context.Context, cclfMap map[string]map[metadataKey][]*cclfFileMetadata) error {
	errCount := 0
	for _, cclfFileMap := range cclfMap {
		for _, cclfFileList := range cclfFileMap {
			for _, cclf := range cclfFileList {
				func() {
					close := metrics.NewChild(ctx, fmt.Sprintf("cleanUpCCLF%d", cclf.cclfNum))
					defer close()

					fmt.Printf("Cleaning up file %s.\n", cclf.filePath)
					log.API.Infof("Cleaning up file %s", cclf.filePath)
					folderName := filepath.Base(cclf.filePath)
					newpath := fmt.Sprintf("%s/%s", conf.GetEnv("PENDING_DELETION_DIR"), folderName)
					if !cclf.imported {
						// check the timestamp on the failed files
						elapsed := time.Since(cclf.deliveryDate).Hours()
						deleteThreshold := utils.GetEnvInt("BCDA_ETL_FILE_ARCHIVE_THRESHOLD_HR", 72)
						if int(elapsed) > deleteThreshold {
							if _, err := os.Stat(newpath); err == nil {
								return
							}
							// move the (un)successful files to the deletion dir
							err := os.Rename(cclf.filePath, newpath)
							if err != nil {
								errCount++
								errMsg := fmt.Sprintf("File %s failed to clean up properly: %v", cclf.filePath, err)
								fmt.Println(errMsg)
								log.API.Error(errMsg)
							} else {
								fmt.Printf("File %s never ingested, moved to the pending deletion dir.\n", cclf.filePath)
								log.API.Infof("File %s never ingested, moved to the pending deletion dir", cclf.filePath)
							}
						}
					} else {
						if _, err := os.Stat(newpath); err == nil {
							return
						}
						// move the successful files to the deletion dir
						err := os.Rename(cclf.filePath, newpath)
						if err != nil {
							errCount++
							errMsg := fmt.Sprintf("File %s failed to clean up properly: %v", cclf.filePath, err)
							fmt.Println(errMsg)
							log.API.Error(errMsg)
						} else {
							fmt.Printf("File %s successfully ingested, moved to the pending deletion dir.\n", cclf.filePath)
							log.API.Infof("File %s successfully ingested, moved to the pending deletion dir", cclf.filePath)
						}
					}
				}()
			}
		}
	}
	if errCount > 0 {
		return fmt.Errorf("%d files could not be cleaned up", errCount)
	}
	return nil
}
