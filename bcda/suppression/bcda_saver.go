package suppression

import (
	"context"

	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/suppression_utils"
)

type BCDASaver struct {
	Repo *postgres.Repository
}

func (saver BCDASaver) SaveFile(suppressionMetaFile suppression_utils.SuppressionFile) (fileID uint, err error) {
	return saver.Repo.CreateSuppressionFile(context.Background(), suppressionMetaFile)
}

func (saver BCDASaver) UpdateImportStatus(metadata suppression_utils.SuppressionFileMetadata, status string) error {
	return saver.Repo.UpdateSuppressionFileImportStatus(context.Background(), metadata.FileID, status)
}

func (saver BCDASaver) SaveSuppression(suppression suppression_utils.Suppression) error {
	return saver.Repo.CreateSuppression(context.Background(), suppression)
}
