package optout

type Saver interface {
	SaveFile(optOutFile OptOutFile) (fileID uint, err error)
	SaveOptOutRecord(optOutRecord OptOutRecord) error
	UpdateImportStatus(metadata OptOutFilenameMetadata, status string) error
}
