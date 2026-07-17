package beneprefs

// type FakeSaver struct {
// 	Files         []BenePrefsFile
// 	BenePrefsRecords []BenePrefsRecord
// }

// func (m *FakeSaver) SaveFile(BenePrefsFile BenePrefsFile) (fileID uint, err error) {
// 	fileID = uint(len(m.Files))
// 	m.Files = append(m.Files, BenePrefsFile)
// 	return fileID, nil
// }

// func (m *FakeSaver) CreateBenePrefsRecord(BenePrefsRecord BenePrefsRecord) error {
// 	m.BenePrefsRecords = append(m.BenePrefsRecords, BenePrefsRecord)
// 	return nil
// }

// func (m *FakeSaver) UpdateImportStatus(metadata BenePrefsFilenameMetadata, status string) error {
// 	m.Files[metadata.FileID].ImportStatus = status
// 	return nil
// }
