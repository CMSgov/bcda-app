package suppression

import (
	"context"

	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/optout"
)

type BCDASaver struct {
	Repo *postgres.Repository
}

func (saver BCDASaver) SaveFile(suppressionMetaFile optout.OptOutFile) (fileID uint, err error) {
	return saver.Repo.CreateSuppressionFile(context.Background(), suppressionMetaFile)
}

func (saver BCDASaver) UpdateImportStatus(metadata optout.OptOutFilenameMetadata, status string) error {
	return saver.Repo.UpdateSuppressionFileImportStatus(context.Background(), metadata.FileID, status)
}

func (saver BCDASaver) SaveOptOutRecord(suppression optout.OptOutRecord) error {
	return saver.Repo.CreateSuppression(context.Background(), suppression)
}
