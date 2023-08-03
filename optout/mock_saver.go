package optout

type MockSaver struct {
	Files         []OptOutFile
	OptOutRecords []OptOutRecord
}

func (m MockSaver) SaveFile(optOutFile OptOutFile) (fileID uint, err error) {
	fileID = uint(len(m.Files))
	m.Files = append(m.Files, optOutFile)
	return fileID, nil
}

func (m MockSaver) SaveOptOutRecord(optOutRecord OptOutRecord) error {
	m.OptOutRecords = append(m.OptOutRecords, optOutRecord)
	return nil
}

func (m MockSaver) UpdateImportStatus(metadata OptOutFilenameMetadata, status string) error {
	file := m.Files[metadata.FileID]
	file.ImportStatus = status
	return nil
}
