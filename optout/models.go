package optout

import "time"

const ImportInprog = "In-Progress"
const ImportComplete = "Completed"
const ImportFail = "Failed"

// An OptOutFile is a basic file representation that can be stored in a database.
type OptOutFile struct {
	ID           uint
	Name         string
	Timestamp    time.Time
	ImportStatus string
}

// OptOutFilenameMetadata is metadata information parsed from the filename.
type OptOutFilenameMetadata struct {
	Name         string
	Timestamp    time.Time
	FilePath     string
	Imported     bool
	DeliveryDate time.Time
	FileID       uint
}

func (m OptOutFilenameMetadata) String() string {
	if m.FilePath != "" {
		return m.FilePath
	}
	return m.Name
}

// An OptOutRecord represents a single record parsed from an opt out file.
type OptOutRecord struct {
	ID                  uint
	FileID              uint
	MBI                 string
	SourceCode          string
	EffectiveDt         time.Time
	PrefIndicator       string
	SAMHSASourceCode    string
	SAMHSAEffectiveDt   time.Time
	SAMHSAPrefIndicator string
	ACOCMSID            string
	BeneficiaryLinkKey  int
}
