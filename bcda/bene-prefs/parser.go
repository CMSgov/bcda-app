package beneprefs

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
)

const (
	mbiStart, mbiEnd                             = 0, 11
	lKeyStart, lKeyEnd                           = 11, 21
	effectiveDtStart, effectiveDtEnd             = 354, 362
	sourceCdeStart, sourceCdeEnd                 = 362, 367
	prefIndtorStart, prefIndtorEnd               = 368, 369
	samhsaEffectiveDtStart, samhsaEffectiveDtEnd = 369, 377
	samhsaSourceCdeStart, samhsaSourceCdeEnd     = 377, 382
	samhsaPrefIndtorStart, samhsaPrefIndtorEnd   = 383, 384
	acoIdStart, acoIdEnd                         = 384, 389
)

// parseMetadata parses the filename into a BenePrefsFilenameMetadata struct, returning an error if the filename is invalid or the date cannot be parsed
func parseMetadata(filename string) (models.BenePrefsFilenameMetadata, error) {
	var metadata models.BenePrefsFilenameMetadata
	matches, isBenePrefs := parseFileName(filename)
	if !isBenePrefs {
		return metadata, fmt.Errorf("invalid filename for file: %s", filename)
	}

	filenameDate := matches[3]
	t, err := time.Parse("D060102.T150405", filenameDate)
	if err != nil || t.IsZero() {
		return metadata, fmt.Errorf("failed to parse date '%s' from file: %s, err: %w", filenameDate, filename, err)
	}

	metadata.Timestamp = t
	metadata.Name = matches[0]

	return metadata, nil
}

// parseFileName parses the given filename into matches and verifies that it is a bene-prefs file via name regex
func parseFileName(filename string) ([]string, bool) {
	filenameRegexp := regexp.MustCompile(`((P|T)\#EFT)\.ON\.ACO\.NGD1800\.DPRF\.(D\d{6}\.T\d{6})\d`)
	matches := filenameRegexp.FindStringSubmatch(filename)
	if len(matches) > 3 {
		return matches, true
	}

	return nil, false
}

// parseRecord parses bytes into a BenePrefsRecord struct, returning an error if any of the fields are invalid
func parseRecord(metadata *models.BenePrefsFilenameMetadata, b []byte) (*models.BenePrefsRecord, error) {
	ds := string(bytes.TrimSpace(b[effectiveDtStart:effectiveDtEnd]))
	dt, err := convertDt(ds)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the effective date '%s' from file: %s, err: %w", ds, metadata.FilePath, err)
	}
	ds = string(bytes.TrimSpace(b[samhsaEffectiveDtStart:samhsaEffectiveDtEnd]))
	samhsaDt, err := convertDt(ds)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the samhsa effective date '%s' from file: %s, err: %w", ds, metadata.FilePath, err)
	}
	keyval := string(bytes.TrimSpace(b[lKeyStart:lKeyEnd]))
	if keyval == "" {
		keyval = "0"
	}
	lk, err := strconv.Atoi(keyval)
	if err != nil {
		return nil, fmt.Errorf("failed to parse beneficiary link key from file: %s, err: %w", metadata.FilePath, err)
	}

	return &models.BenePrefsRecord{
		FileID:              metadata.FileID,
		MBI:                 string(bytes.TrimSpace(b[mbiStart:mbiEnd])),
		SourceCode:          string(bytes.TrimSpace(b[sourceCdeStart:sourceCdeEnd])),
		EffectiveDt:         dt,
		PrefIndicator:       string(bytes.TrimSpace(b[prefIndtorStart:prefIndtorEnd])),
		SAMHSASourceCode:    string(bytes.TrimSpace(b[samhsaSourceCdeStart:samhsaSourceCdeEnd])),
		SAMHSAEffectiveDt:   samhsaDt,
		SAMHSAPrefIndicator: string(bytes.TrimSpace(b[samhsaPrefIndtorStart:samhsaPrefIndtorEnd])),
		BeneficiaryLinkKey:  lk,
		ACOCMSID:            string(bytes.TrimSpace(b[acoIdStart:acoIdEnd])),
	}, nil
}

func convertDt(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse("20060102", s)
	if err != nil || t.IsZero() {
		return t, err
	}
	return t, nil
}
