package suppression_utils

type Saver interface {
	SaveFile(suppressionMetaFile SuppressionFile) (fileID uint, err error)
	SaveSuppression(suppression Suppression) error
	UpdateImportStatus(metadata SuppressionFileMetadata, status string) error
}
