package optout

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
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

func ParseMetadata(filename string) (OptOutFilenameMetadata, error) {
	var metadata OptOutFilenameMetadata
	isOptOut, matches := IsOptOut(filename)
	if !isOptOut {
		fmt.Printf("Invalid filename for file: %s.\n", filename)
		err := fmt.Errorf("invalid filename for file: %s", filename)
		return metadata, err
	}

	filenameDate := matches[3]
	t, err := time.Parse("D060102.T150405", filenameDate)
	if err != nil || t.IsZero() {
		fmt.Printf("Failed to parse date '%s' from file: %s.\n", filenameDate, filename)
		err = errors.Wrapf(err, "failed to parse date '%s' from file: %s", filenameDate, filename)
		return metadata, err
	}

	metadata.Timestamp = t
	metadata.Name = matches[0]

	return metadata, nil
}

func IsOptOut(filename string) (isOptOut bool, matches []string) {
	filenameRegexp := regexp.MustCompile(`((P|T)\#EFT)\.ON\.ACO\.NGD1800\.DPRF\.(D\d{6}\.T\d{6})\d`)
	matches = filenameRegexp.FindStringSubmatch(filename)
	if len(matches) > 3 {
		isOptOut = true
	}
	return isOptOut, matches
}

func ParseRecord(metadata *OptOutFilenameMetadata, b []byte) (*OptOutRecord, error) {
	ds := string(bytes.TrimSpace(b[effectiveDtStart:effectiveDtEnd]))
	dt, err := ConvertDt(ds)
	if err != nil {
		fmt.Printf("Failed to parse the effective date '%s' from file: %s.\n", ds, metadata.FilePath)
		err = errors.Wrapf(err, "failed to parse the effective date '%s' from file: %s", ds, metadata.FilePath)
		return nil, err
	}
	ds = string(bytes.TrimSpace(b[samhsaEffectiveDtStart:samhsaEffectiveDtEnd]))
	samhsaDt, err := ConvertDt(ds)
	if err != nil {
		fmt.Printf("Failed to parse the samhsa effective date '%s' from file: %s.\n", ds, metadata.FilePath)
		err = errors.Wrapf(err, "failed to parse the samhsa effective date '%s' from file: %s", ds, metadata.FilePath)
		return nil, err
	}
	keyval := string(bytes.TrimSpace(b[lKeyStart:lKeyEnd]))
	if keyval == "" {
		keyval = "0"
	}
	lk, err := strconv.Atoi(keyval)
	if err != nil {
		fmt.Printf("Failed to parse beneficiary link key from file: %s.\n", metadata.FilePath)
		err = errors.Wrapf(err, "failed to parse beneficiary link key from file: %s", metadata.FilePath)
		return nil, err
	}

	return &OptOutRecord{
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

func ConvertDt(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse("20060102", s)
	if err != nil || t.IsZero() {
		return t, err
	}
	return t, nil
}
