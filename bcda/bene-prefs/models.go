package beneprefs

// const ImportInprog = "In-Progress"
// const ImportComplete = "Completed"
// const ImportFail = "Failed"

// // An BenePrefsFile is a basic file representation that can be stored in a database.
// type BenePrefsFile struct {
// 	ID           uint
// 	Name         string
// 	Timestamp    time.Time
// 	ImportStatus string
// }

// // BenePrefsFilenameMetadata is metadata information parsed from the filename.
// type BenePrefsFilenameMetadata struct {
// 	Name         string
// 	Timestamp    time.Time
// 	FilePath     string
// 	Imported     bool
// 	DeliveryDate time.Time
// 	FileID       uint
// }

// func (m BenePrefsFilenameMetadata) String() string {
// 	if m.FilePath != "" {
// 		return m.FilePath
// 	}
// 	return m.Name
// }

// // An BenePrefsRecord represents a single record parsed from an opt out file.
// type BenePrefsRecord struct {
// 	ID                  uint
// 	FileID              uint
// 	MBI                 string
// 	SourceCode          string
// 	EffectiveDt         time.Time
// 	PrefIndicator       string
// 	SAMHSASourceCode    string
// 	SAMHSAEffectiveDt   time.Time
// 	SAMHSAPrefIndicator string
// 	ACOCMSID            string
// 	BeneficiaryLinkKey  int
// }
