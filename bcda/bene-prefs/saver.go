package beneprefs

// Out opt savers save file metadata and individual records into the database.
// type Saver interface {
// 	SaveFile(BenePrefsFile BenePrefsFile) (fileID uint, err error)
// 	CreateBenePrefsRecord(BenePrefsRecord BenePrefsRecord) error
// 	UpdateImportStatus(metadata BenePrefsFilenameMetadata, status string) error
// }

// type Saver struct {
// 	Repo *postgres.Repository
// }

// func (saver *Saver) SaveFile(suppressionMetaFile BenePrefsFile) (fileID uint, err error) {
// 	return saver.Repo.CreateSuppressionFile(context.Background(), suppressionMetaFile)
// }

// func (saver *Saver) UpdateImportStatus(metadata BenePrefsFilenameMetadata, status string) error {
// 	return saver.Repo.UpdateSuppressionFileImportStatus(context.Background(), metadata.FileID, status)
// }

// func (saver *Saver) CreateBenePrefsRecord(suppression BenePrefsRecord) error {
// 	return saver.Repo.CreateSuppression(context.Background(), suppression)
// }
