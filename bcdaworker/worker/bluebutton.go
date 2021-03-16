package worker

import (
	"encoding/json"
	"strings"

	"github.com/CMSgov/bcda-app/bcda/client"
	models "github.com/CMSgov/bcda-app/bcda/models"
	"github.com/pkg/errors"
)

// This method will ensure that a valid BlueButton ID is returned.
// If you use cclfBeneficiary.BlueButtonID you will not be guaranteed a valid value
func getBlueButtonID(bb client.APIClient, mbi string) (blueButtonID string, err error) {
	hashedIdentifier := client.HashIdentifier(mbi)
	jsonData, err := bb.GetPatientByIdentifierHash(hashedIdentifier)
	if err != nil {
		return "", err
	}

	var patient models.Patient
	err = json.Unmarshal([]byte(jsonData), &patient)
	if err != nil {
		return "", err
	}

	if len(patient.Entry) == 0 {
		err = errors.New("patient identifier not found at Blue Button for CCLF")
		return "", err
	}

	var foundIdentifier = false
	var foundBlueButtonID = false
	blueButtonID = patient.Entry[0].Resource.ID
	for _, identifier := range patient.Entry[0].Resource.Identifier {
		if strings.Contains(identifier.System, "us-mbi") {
			if identifier.Value == mbi {
				foundIdentifier = true
			}
		} else if strings.Contains(identifier.System, "bene_id") && identifier.Value == blueButtonID {
			foundBlueButtonID = true
		} else if strings.EqualFold(identifier.System, "http://terminology.hl7.org/CodeSystem/v2-0203") {
			// This hot-fix logic to handle the changes made in this PR: 
			// https://github.com/CMSgov/beneficiary-fhir-data/pull/474
			// Specifically:
			// https://github.com/CMSgov/beneficiary-fhir-data/pull/474/files#diff-97195cabdd2698fa9148e9ad32fb8fef8dd462a55dabb9eaf4a4b4300f691fddL112
			// https://github.com/CMSgov/beneficiary-fhir-data/pull/474/files#diff-97195cabdd2698fa9148e9ad32fb8fef8dd462a55dabb9eaf4a4b4300f691fddR132
			// https://github.com/CMSgov/beneficiary-fhir-data/pull/474/files#diff-97195cabdd2698fa9148e9ad32fb8fef8dd462a55dabb9eaf4a4b4300f691fddL191
			if identifier.Value == mbi {
				foundIdentifier = true
				foundBlueButtonID = true
			}
		}
	}
	if !foundIdentifier {
		err = errors.New("Identifier not found")
		return "", err
	}
	if !foundBlueButtonID {
		err = errors.New("Blue Button identifier not found in the identifiers")
		return "", err
	}

	return blueButtonID, nil
}
