package models

// CCLFBeneficiaryService contains methods to interact with CCLF Beneficiary data
type CCLFBeneficiaryService interface {
	// GetNewAndExistingBeneficiaries, when supplied with the "since" parameter, returns two arrays
	// the first array contains all NEW beneficaries that were added to CCLF since the date supplied
	// the second array contains all EXISTING benficiaries that have existed in CCLF since prior to the date supplied
	GetNewAndExistingBeneficiaries(includeSuppressed bool, since string) (newBeneficiaries, beneficiaries []*CCLFBeneficiary, err error)

	// GetBeneficiaries retrieves all beneficiaries associated with the ACO, contained in one array
	GetBeneficiaries(includeSuppressed bool) ([]bcda/models/service.go*CCLFBeneficiary, error)
}
