/* #nosec G404 */
package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/text/language"

	"golang.org/x/text/cases"
)

type field struct {
	l int
	v string
}

type record struct {
	hicn, blk                                    field
	firstName, middleName, lastName              field
	dob                                          field
	addr1, addr2, addr3, city, state, zip5, zip4 field
	gender                                       field
	encDate                                      field
	effDate, srcCode, mechCode, prefInd          field
	saEffDate, saSrcCode, saMechCode, saPrefInd  field
	acoID, acoName                               field
}

func main() {
	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	now := time.Now()

	// 1-800-MEDICARE filename format: (T|P)#EFT.ON.ACO.NGD1800.DPRF.Dyymmdd.Thhmmsst
	fileName := fmt.Sprintf("T#EFT.ON.ACO.NGD1800.DPRF.%s", now.Format("D060102.T1504050"))
	outf, err := os.Create(filepath.Clean(fileName))
	if err != nil {
		panic(err)
	}

	_, err = outf.WriteString(fmt.Sprintf("HDR_BENEDATASHR%s\n", now.Format("20060102")))
	if err != nil {
		panic(err)
	}

	recCount := 0

	for _, acoID := range []string{"A9990", "A9991", "A9992", "A9993", "A9994"} {
		inf, err := os.Open(fmt.Sprintf("%s-HICNs", acoID))
		if err != nil {
			panic(err)
		}

		s := bufio.NewScanner(inf)

		for s.Scan() {
			// Randomly select HICNs from ACO for suppression records
			if r.Intn(2) == 0 {
				continue
			}

			hicn := string(s.Bytes())
			p := profile(hicn, acoID)
			recs := records(p)
			for _, r := range recs {
				for _, f := range []field{r.hicn, r.blk, r.firstName, r.middleName, r.lastName, r.dob, r.addr1, r.addr2, r.addr3, r.city, r.state, r.zip5, r.zip4, r.gender, r.encDate, r.effDate, r.srcCode, r.mechCode, r.prefInd, r.saEffDate, r.saSrcCode, r.saMechCode, r.saPrefInd, r.acoID, r.acoName} {
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

		_, err = outf.WriteString(fmt.Sprintf("TRL_BENEDATASHR%s%-10d", now.Format("20060102"), recCount))
		if err != nil {
			panic(err)
		}

		err = inf.Close()
		if err != nil {
			panic(err)
		}
	}

	err = outf.Close()
	if err != nil {
		panic(err)
	}
}

func profile(hicn string, acoID string) record {
	dobMin, _ := time.Parse("2006-01-02", "1900-01-01")
	dobMax := time.Now().Add(-65 * 365 * 24 * time.Hour)

	p := record{
		hicn:       field{l: 11, v: hicn},                     // HICN
		blk:        field{l: 10},                              // Beneficiary link key
		firstName:  field{l: 30, v: randWord(1, 30)},          // Beneficiary first name
		middleName: field{l: 30},                              // Beneficiary middle name
		lastName:   field{l: 40, v: randWord(1, 30)},          // Beneficiary last name
		dob:        field{l: 8, v: ccyymmdd(dobMin, dobMax)},  // Beneficiary date of birth
		addr1:      field{l: 55, v: addr1()},                  // Beneficiary address line 1
		addr2:      field{l: 55},                              // Beneficiary address line 2
		addr3:      field{l: 55},                              // Beneficiary address line 3
		city:       field{l: 40, v: randWord(1, 40)},          // Beneficiary city
		state:      field{l: 2, v: "ST"},                      // Beneficiary state
		zip5:       field{l: 5, v: "00000"},                   // Beneficiary first five digits of ZIP code
		zip4:       field{l: 4},                               // Beneficiary last four digits of ZIP code
		gender:     field{l: 1, v: oneOfStr("M", "F", "U")},   // Beneficiary gender
		acoID:      field{l: 5, v: acoID},                     // ACO identifier
		acoName:    field{l: 70, v: randWord(1, 66) + " ACO"}, // ACO legal name
	}

	return p
}

func records(profile record) []record {
	// Create 0-3 suppression records for this profile
	ct := rand.Intn(4)
	encDtMin, encDtMax := time.Now().Add(-7*24*time.Hour), time.Now()
	effDtMin, effDtMax := time.Now().Add(-7*24*time.Hour), time.Now().Add(3*24*time.Hour)
	var records []record
	for i := 0; i < ct; i++ {
		r := profile

		r.srcCode = field{l: 5, v: oneOfStr("1800", "")}    // Beneficiary data sharing source code
		r.saSrcCode = field{l: 5, v: oneOfStr("1-800", "")} // Beneficiary substance abuse data sharing source code
		if r.srcCode.v == "" && r.saSrcCode.v == "" {
			// One or both types of data sharing should have values
			continue
		}

		r.encDate = field{l: 8, v: ccyymmdd(encDtMin, encDtMax)} // Encounter date

		r.effDate = field{l: 8, v: ""}  // Beneficiary data sharing effective date
		r.mechCode = field{l: 1, v: ""} // Beneficiary option data sharing decision mechanism code
		r.prefInd = field{l: 1, v: ""}  // Beneficiary data sharing preference indicator
		// If beneficiary data sharing source code is populated, also populate its associated fields (ICD v9.1 p.11)
		if r.srcCode.v != "" {
			r.effDate.v = ccyymmdd(effDtMin, effDtMax)
			r.mechCode.v = "T"
			r.prefInd.v = oneOfStr("Y", "N")
		}

		r.saEffDate = field{l: 8, v: ""}  // Beneficiary substance abuse data sharing effective date
		r.saMechCode = field{l: 1, v: ""} // Beneficiary option substance abuse decision mechanism code
		r.saPrefInd = field{l: 1, v: ""}  // Beneficiary substance abuse data sharing preference indicator
		// If beneficiary substance abuse data sharing source code is populated, also populate its associated fields (ICD v9.1 p.11)
		if r.saSrcCode.v != "" {
			r.saEffDate.v = ccyymmdd(effDtMin, effDtMax)
			r.saMechCode.v = "T"
			r.saPrefInd.v = "N"
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

func randWord(minLen, maxLen int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	l := rand.Intn(maxLen-minLen) + minLen
	w := ""
	for i := 0; i < l; i++ {
		w = w + string(letters[rand.Intn(len(letters))])
	}
	return cases.Title(language.Und).String(w)
}

func addr1() string {
	return fmt.Sprintf("%d %s %s", rand.Intn(50000), randWord(1, 44), oneOfStr("St", "Ave", "Dr", "Blvd", "Way"))
}
