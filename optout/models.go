package optout

import "time"

const ImportInprog = "In-Progress"
const ImportComplete = "Completed"
const ImportFail = "Failed"

type OptOutFile struct {
	ID           uint
	Name         string
	Timestamp    time.Time
	ImportStatus string
}

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
