package optout

type MockSaver struct {
	files        []SuppressionFile
	suppressions []Suppression
}

func (m MockSaver) SaveFile(suppressionFile SuppressionFile) (fileID uint, err error) {
	fileID = uint(len(m.files))
	append(m.files, suppressionFile)
	return fileID, nil
}

func (m MockSaver) SaveSuppression(suppression Suppression) error {
	append(m.suppressions, suppression)
	return nil
}

func (m MockSaver) UpdateImportStatus(metadata SuppressionFileMetadata, status string) error {
	file := m.files[metadata.FileID]
	file.ImportStatus = status
	return nil
}
