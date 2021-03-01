package csv

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/sirupsen/logrus"
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

	const layout = "01/02/2006"
	t, err := time.Parse(layout, birth)
	if err != nil {
		logrus.Warnf("Could not parse birth date %s %s. Will leave value unset.",
			birth, err.Error())
		return
	}
	a.BeneDOB = t
}

func (a *alr) setDeath(death string) {
	if len(death) == 0 {
		return
	}

	const layout = "01/02/2006"
	t, err := time.Parse(layout, death)
	if err != nil {
		logrus.Warnf("Could not parse death date %s %s. Will leave value unset.",
			death, err.Error())
		return
	}
	a.BeneDOD = t
}
