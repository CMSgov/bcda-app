package beneprefs

// import (
// 	"context"

// 	bp "github.com/CMSgov/bcda-app/bcda/bene-prefs"
// 	"github.com/CMSgov/bcda-app/bcda/models/postgres"
// )

// type BCDASaver struct {
// 	Repo *postgres.Repository
// }

// func (saver *BCDASaver) SaveFile(suppressionMetaFile bp.BenePrefsFile) (fileID uint, err error) {
// 	return saver.Repo.CreateSuppressionFile(context.Background(), suppressionMetaFile)
// }

// func (saver *BCDASaver) UpdateImportStatus(metadata bp.BenePrefsFilenameMetadata, status string) error {
// 	return saver.Repo.UpdateSuppressionFileImportStatus(context.Background(), metadata.FileID, status)
// }

// func (saver *BCDASaver) CreateBenePrefsRecord(suppression bp.BenePrefsRecord) error {
// 	return saver.Repo.CreateSuppression(context.Background(), suppression)
// }
