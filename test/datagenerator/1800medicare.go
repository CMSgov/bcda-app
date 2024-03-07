/* #nosec G404 */
package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/text/language"

	"github.com/Pallinder/go-randomdata"
	"golang.org/x/text/cases"
)

type field struct {
	l int
	v string
}

type record struct {
	mbi                 field
	firstName, lastName field
	dob                 field
	effDate, prefInd    field
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	now := time.Now()

	// These filenames match the expected filenames that NGD receives. These are NOT the same as the filenames that we send as AB2D/DPC.
	//
	// AB2D/DPC SEND 1-800-MEDICARE filename format: P#EFT.ON.<TEAM>.NGD.REQ.Dyymmdd.Thhmmsst
	// NGD RECEIVE 1-800-MEDICARE filename format: P.<TEAM>.NGD.REQ.Dyymmdd.Thhmmsst.OUT
	//
	reqFileName := fmt.Sprintf("P.DPC.NGD.REQ.%s.OUT", now.Format("D060102.T1504050"))
	reqAb2dFileName := fmt.Sprintf("P.AB2D.NGD.REQ.%s.OUT", now.Format("D060102.T1504050"))
	confFileName := fmt.Sprintf("P.DPC.NGD.CONF.%s.OUT", now.Format("D060102.T1504050"))

	reqOutf, err := os.Create(filepath.Clean(reqFileName))

	if err != nil {
		panic(err)
	}

	reqAb2dOutf, err := os.Create(filepath.Clean(reqAb2dFileName))

	if err != nil {
		panic(err)
	}

	confOutf, err := os.Create(filepath.Clean(confFileName))

	if err != nil {
		panic(err)
	}

	_, err = reqOutf.WriteString(fmt.Sprintf("HDR_BENEDATAREQ%s\n", now.Format("20060102")))

	if err != nil {
		panic(err)
	}

	_, err = reqAb2dOutf.WriteString(fmt.Sprintf("HDR_BENEDATAREQ%s\n", now.Format("20060102")))

	if err != nil {
		panic(err)
	}

	_, err = confOutf.WriteString(fmt.Sprintf("HDR_BENECONFIRM%s\n", now.Format("20060102")))

	if err != nil {
		panic(err)
	}

	reqRecCount := 0
	reqAb2dRecCount := 0
	confRecCount := 0

	// AB2D
	// Generate request file.
	//
	numAb2dReqRecords := 10
	for i := 0; i < numAb2dReqRecords; i++ {
		mbi := randMbi()
		p := profile(mbi)
		recs := records(p)
		for _, r := range recs {
			for _, f := range []field{r.mbi, r.effDate, r.prefInd} {
				_, err = reqAb2dOutf.WriteString(fmt.Sprintf("%-"+fmt.Sprint(f.l)+"s", f.v))
				if err != nil {
					panic(err)
				}
			}

			_, err = reqAb2dOutf.WriteString("\n")
			if err != nil {
				panic(err)
			}

		}
		reqAb2dRecCount += 1
	}

	_, err = reqAb2dOutf.WriteString(fmt.Sprintf("TRL_BENEDATAREQ%s%010d", now.Format("20060102"), reqAb2dRecCount))

	if err != nil {
		panic(err)
	}
	err = reqAb2dOutf.Close()
	if err != nil {
		panic(err)
	}

	// DPC
	// Generate request and confirmation file.
	//
	numDpcReqRecords := 10
	for i := 0; i < numDpcReqRecords; i++ {
		mbi := randMbi()
		p := profile(mbi)
		recs := records(p)
		for _, r := range recs {
			for _, f := range []field{r.mbi, r.firstName, r.lastName, r.dob, r.effDate, r.prefInd} {
				_, err = reqOutf.WriteString(fmt.Sprintf("%-"+fmt.Sprint(f.l)+"s", f.v))
				if err != nil {
					panic(err)
				}
			}

			if r.prefInd.v != "" {
				confRecCount += 1

				for _, f := range []field{r.mbi, r.effDate, r.prefInd, {l: 10, v: "Accepted"}, {l: 2, v: "00"}} {
					_, err = confOutf.WriteString(fmt.Sprintf("%-"+fmt.Sprint(f.l)+"s", f.v))
					if err != nil {
						panic(err)
					}
				}
				_, err = confOutf.WriteString("\n")
				if err != nil {
					panic(err)
				}
			}
			_, err = reqOutf.WriteString("\n")
			if err != nil {
				panic(err)
			}

		}
		reqRecCount += 1
	}

	_, err = reqOutf.WriteString(fmt.Sprintf("TRL_BENEDATAREQ%s%010d", now.Format("20060102"), reqRecCount))

	if err != nil {
		panic(err)
	}

	_, err = confOutf.WriteString(fmt.Sprintf("TRL_BENECONFIRM%s%010d", now.Format("20060102"), confRecCount))

	if err != nil {
		panic(err)
	}

	err = reqOutf.Close()
	if err != nil {
		panic(err)
	}

	err = confOutf.Close()
	if err != nil {
		panic(err)
	}
}

func profile(mbi string) record {
	dobMin, _ := time.Parse("2006-01-02", "1900-01-01")
	dobMax := time.Now().Add(-65 * 365 * 24 * time.Hour)

	p := record{
		mbi:       field{l: 11, v: mbi},                                           // mbi
		firstName: field{l: 30, v: randomdata.FirstName(randomdata.RandomGender)}, // Beneficiary first name
		lastName:  field{l: 40, v: randomdata.LastName()},                         // Beneficiary last name
		dob:       field{l: 8, v: ccyymmdd(dobMin, dobMax)},                       // Beneficiary date of birth
	}

	return p
}

func records(profile record) []record {
	// Create 0-1 suppression records for this profile
	// ct := rand.Intn(2)
	records := []record{}

	for i := 0; i < 1; i++ {
		r := profile

		is1800 := oneOfStr("1800", "")
		r.effDate = field{l: 8, v: ""} // Beneficiary data sharing effective date
		r.prefInd = field{l: 1, v: ""} // Beneficiary data sharing preference indicator

		// If beneficiary data sharing source code is populated, also populate its associated fields (ICD v9.1 p.11)
		if is1800 != "" {
			effDtMin, effDtMax := time.Now().Add(-7*24*time.Hour), time.Now().Add(3*24*time.Hour)
			r.effDate.v = ccyymmdd(effDtMin, effDtMax)
			r.prefInd.v = oneOfStr("Y", "N")
		}

		records = append(records, r)
	}

	return records
}

func ccyymmdd(min, max time.Time) string {
	diffHrs := max.Sub(min).Hours()
	dt := min.Add(time.Duration(rand.Float64()*diffHrs) * time.Hour)
	return dt.Format("20060102")
}

func oneOfStr(strs ...string) string {
	return strs[rand.Intn(len(strs))]
}

func randMbi() string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	const digits = "0123456789"
	w := string(letters[rand.Intn(len(letters))])
	for i := 0; i < 10; i++ {
		w = w + string(digits[rand.Intn(len(digits))])
	}
	return cases.Title(language.Und).String(w)
}
