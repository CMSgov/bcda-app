package optout

// Out opt savers save file metadata and individual records into the database.
type Saver interface {
	SaveFile(optOutFile OptOutFile) (fileID uint, err error)
	SaveOptOutRecord(optOutRecord OptOutRecord) error
	UpdateImportStatus(metadata OptOutFilenameMetadata, status string) error
}
