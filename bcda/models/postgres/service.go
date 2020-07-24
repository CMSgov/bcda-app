package postgres

import "github.com/CMSgov/bcda-app/bcda/models"

type Service struct {
	repository *Repository
}

func (s *Service) GetNewAndExistingBeneficiaries(includeSuppressed bool, since string) (newBeneficiaries, beneficiaries []*models.CCLFBeneficiary, err error) {
	return nil, nil, nil
}
