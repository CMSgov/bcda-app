package optout

type MockSaver struct {
	Files        []SuppressionFile
	Suppressions []Suppression
}

func (m MockSaver) SaveFile(suppressionFile SuppressionFile) (fileID uint, err error) {
	fileID = uint(len(m.Files))
	append(m.Files, suppressionFile)
	return fileID, nil
}

func (m MockSaver) SaveSuppression(suppression Suppression) error {
	append(m.Suppressions, suppression)
	return nil
}

func (m MockSaver) UpdateImportStatus(metadata SuppressionFileMetadata, status string) error {
	file := m.Files[metadata.FileID]
	file.ImportStatus = status
	return nil
}
