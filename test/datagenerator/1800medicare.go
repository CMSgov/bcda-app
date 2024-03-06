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
	hicn                            field
	firstName, middleName, lastName field
	dob                             field
	effDate, prefInd                field
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	now := time.Now()

	// 1-800-MEDICARE filename format: (T|P)#EFT.ON.ACO.NGD1800.DPRF.Dyymmdd.Thhmmsst
	fileName := fmt.Sprintf("P#EFT.ON.DPC.NGD.REQ.%s", now.Format("D060102.T1504050"))
	outf, err := os.Create(filepath.Clean(fileName))
	if err != nil {
		panic(err)
	}

	_, err = outf.WriteString(fmt.Sprintf("HDR_BENEDATAREQ%s\n", now.Format("20060102")))
	if err != nil {
		panic(err)
	}

	recCount := 0

	for i := 0; i < 10; i++ {
		hicn := randMbi()
		p := profile(hicn)
		recs := records(p)
		for _, r := range recs {
			for _, f := range []field{r.hicn, r.firstName, r.middleName, r.lastName, r.dob, r.effDate, r.prefInd} {
				_, err = outf.WriteString(fmt.Sprintf("%-"+fmt.Sprint(f.l)+"s", f.v))
				if err != nil {
					panic(err)
				}
			}
			_, err = outf.WriteString("\n")
			if err != nil {
				panic(err)
			}
		}
		recCount += len(recs)
	}

	_, err = outf.WriteString(fmt.Sprintf("TRL_BENEDATAREQ%s%-10d", now.Format("20060102"), recCount))
	if err != nil {
		panic(err)
	}

	err = outf.Close()
	if err != nil {
		panic(err)
	}
}

func profile(hicn string) record {
	dobMin, _ := time.Parse("2006-01-02", "1900-01-01")
	dobMax := time.Now().Add(-65 * 365 * 24 * time.Hour)

	p := record{
		hicn:      field{l: 11, v: hicn},                                          // HICN
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
