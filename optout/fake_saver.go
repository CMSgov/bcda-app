package optout

type FakeSaver struct {
	Files         []OptOutFile
	OptOutRecords []OptOutRecord
}

func (m *FakeSaver) SaveFile(optOutFile OptOutFile) (fileID uint, err error) {
	fileID = uint(len(m.Files))
	m.Files = append(m.Files, optOutFile)
	return fileID, nil
}

func (m *FakeSaver) SaveOptOutRecord(optOutRecord OptOutRecord) error {
	m.OptOutRecords = append(m.OptOutRecords, optOutRecord)
	return nil
}

func (m *FakeSaver) UpdateImportStatus(metadata OptOutFilenameMetadata, status string) error {
	m.Files[metadata.FileID].ImportStatus = status
	return nil
}
