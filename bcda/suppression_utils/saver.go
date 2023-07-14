package suppression_utils

type Saver interface {
	SaveFile(suppressionMetaFile SuppressionFile) (fileID uint, err error)
	SaveSuppression(suppression Suppression) error
	UpdateImportStatus(fileID uint, status string) error
}
