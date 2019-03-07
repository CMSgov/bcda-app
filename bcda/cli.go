package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
)

func createACO(name, cmsID string) (string, error) {
	if name == "" {
		return "", errors.New("ACO name (--name) must be provided")
	}

	var cmsIDPt *string
	// ACO ID is not required, but must match `AXXXX` if provided
	if cmsID != "" {
		acoIDFmt := regexp.MustCompile(`^A\d{4}$`)
		if !acoIDFmt.MatchString(cmsID) {
			return "", errors.New("ACO CMS ID (--cms-id) is invalid")
		}
		cmsIDPt = &cmsID
	}

	acoUUID, err := models.CreateACO(name, cmsIDPt)
	if err != nil {
		return "", err
	}

	return acoUUID.String(), nil
}

type cclfFileMetadata struct {
	env       string
	acoID     string
	cclfNum   int
	timestamp time.Time
}

func importCCLF8(filePath string) error {
	if filePath == "" {
		return errors.New("file path (--file) must be provided")
	}

	fileMetadata, err := getCCLFFileMetadata(filePath)
	if err != nil {
		return err
	}

	if fileMetadata.cclfNum != 8 {
		return errors.New("invalid CCLF8 filename")
	}

	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return err
	}

	fmt.Printf("File contains %s data for ACO %s at %s.\n", fileMetadata.env, fileMetadata.acoID, fileMetadata.timestamp)

	const (
		mbiStart, mbiEnd   = 0, 11
		hicnStart, hicnEnd = 11, 22
	)

	sc := bufio.NewScanner(file)
	for sc.Scan() {
		b := sc.Bytes()
		if len(bytes.TrimSpace(b)) > 0 {
			fmt.Printf("\nMBI: %s\n", b[mbiStart:mbiEnd])
			fmt.Printf("HICN: %s\n", b[hicnStart:hicnEnd])
		}
	}

	return nil
}

func importCCLF9(filePath string) error {
	if filePath == "" {
		return errors.New("file path (--file) must be provided")
	}

	fileMetadata, err := getCCLFFileMetadata(filePath)
	if err != nil {
		return err
	}

	if fileMetadata.cclfNum != 9 {
		return errors.New("invalid CCLF9 filename")
	}

	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return err
	}

	fmt.Printf("File contains %s data for ACO %s at %s.\n", fileMetadata.env, fileMetadata.acoID, fileMetadata.timestamp)

	const (
		currIDStart, currIDEnd               = 1, 12
		prevIDStart, prevIDEnd               = 12, 23
		prevIDEffDateStart, prevIDEffDateEnd = 23, 33
		prevIDObsDateStart, prevIDObsDateEnd = 33, 43
	)

	sc := bufio.NewScanner(file)
	for sc.Scan() {
		b := sc.Bytes()
		if len(bytes.TrimSpace(b)) > 0 {
			fmt.Printf("\nXREF: %s\n", b[0:1])
			fmt.Printf("Current identifier: %s\n", b[currIDStart:currIDEnd])
			fmt.Printf("Previous identifier: %s\n", b[prevIDStart:prevIDEnd])
			fmt.Printf("Previous identifier effective date: %s\n", b[prevIDEffDateStart:prevIDEffDateEnd])
			fmt.Printf("Previous identifier obsolete date: %s\n", b[prevIDObsDateStart:prevIDObsDateEnd])
		}
	}

	return nil
}

func getCCLFFileMetadata(filePath string) (cclfFileMetadata, error) {
	var metadata cclfFileMetadata
	// CCLF8/9 filename convention for SSP: P.A****.ACO.ZC*Y**.Dyymmdd.Thhmmsst
	// Prefix: T = test, P = prod; A**** = ACO ID; ZC* = CCLF file number; Y** = performance year
	filenameRegexp := regexp.MustCompile(`(T|P)\.(A\d{4})\.ACO\.ZC(8|9)Y\d{2}\.(D\d{6}\.T\d{6})\d`)
	filenameMatches := filenameRegexp.FindStringSubmatch(filePath)
	if len(filenameMatches) < 5 {
		return metadata, errors.New("invalid filename")
	}

	filenameDate := filenameMatches[4]
	t, err := time.Parse("D060102.T150405", filenameDate)
	if err != nil {
		return metadata, fmt.Errorf("failed to parse date '%s' from filename", filenameDate)
	}

	cclfNum, err := strconv.Atoi(filenameMatches[3])
	if err != nil {
		return metadata, err
	}

	if filenameMatches[1] == "T" {
		metadata.env = "test"
	} else if filenameMatches[1] == "P" {
		metadata.env = "production"
	}

	metadata.cclfNum = cclfNum
	metadata.acoID = filenameMatches[2]
	metadata.timestamp = t

	return metadata, nil
}
