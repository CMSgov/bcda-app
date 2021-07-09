package csv

/******************************************************************************
This package is responsible for data wrangling and ingesting ALR data.
Contents:
1. alr.go
2. csv.go
Dependencies:
1. models/alr.go
******************************************************************************/

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/log"
)

// Wrap models.Alr to allow us to create setter that will allow us to incrementally build
// an ALR from a CSV file
type alr struct {
	models.Alr
}

func (a *alr) setMBI(mbi string) {
	a.BeneMBI = mbi
}

func (a *alr) setHIC(hic string) {
	a.BeneHIC = hic
}

func (a *alr) setFirstName(firstName string) {
	a.BeneFirstName = firstName
}

func (a *alr) setLastName(lastName string) {
	a.BeneLastName = lastName
}

func (a *alr) setSex(sex string) {
	a.BeneSex = sex
}

func (a *alr) setBirth(birth string) {
	if len(birth) == 0 {
		return
	}

	t, err := time.Parse("01/02/2006", birth)
	if err != nil {
		log.API.Warnf("Could not parse birth date %s %s. Will leave value unset.",
			birth, err.Error())
		return
	}
	a.BeneDOB = t
}

func (a *alr) setDeath(death string) {
	if len(death) == 0 {
		return
	}

	t, err := time.Parse("01/02/2006", death)
	if err != nil {
		log.API.Warnf("Could not parse death date %s %s. Will leave value unset.",
			death, err.Error())
		return
	}
	a.BeneDOD = t
}
