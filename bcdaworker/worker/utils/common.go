package utils

import (
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr"
)

func AlrSlicer(alrModels []*models.Alr, c chan *alr.AlrFhirBulk, limit int, bbbasepath string) {
	for len(alrModels) > limit {
		alrModelsSub := alrModels[:limit]
		fhirBulk := alr.ToFHIR(alrModelsSub, bbbasepath) // Removed timestamp, but can be added back here
		alrModels = alrModels[limit:]

		if fhirBulk == nil {
			continue
		}

		c <- fhirBulk
	}

	// There is one more iteration before we have traverse the whole slice
	fhirBulk := alr.ToFHIR(alrModels, bbbasepath)

	if fhirBulk != nil {
		c <- fhirBulk
	}

	// close channel c since we are no longer writing to it
	close(c)
}
