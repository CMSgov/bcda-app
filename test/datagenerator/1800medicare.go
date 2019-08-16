package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"time"
)

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

type field struct {
	l int
	v string
}

func main() {
	acoIDs := []string{ /*"A9990", "A9991", "A9992", "A9993", */ "A9994"}
	for _, acoID := range acoIDs {
		file, _ := os.Open(fmt.Sprintf("%s-HICNs", acoID))
		s := bufio.NewScanner(file)
		for s.Scan() {
			// Randomly select HICNs from ACO
			if rand.Intn(2) == 0 {
				continue
			}
			hicn := string(s.Bytes())
			p := makeProfile(hicn, acoID)
			records := makeRecords(p)
			for _, r := range records {
				for _, f := range []field{r.hicn, r.blk, r.firstName, r.middleName, r.lastName, r.dob, r.addr1, r.addr2, r.addr3, r.city, r.state, r.zip5, r.zip4, r.gender, r.encDate, r.effDate, r.srcCode, r.mechCode, r.prefInd, r.saEffDate, r.saSrcCode, r.saMechCode, r.saPrefInd, r.acoID, r.acoName} {
					fmt.Printf("%-"+fmt.Sprint(f.l)+"s", f.v)
				}
			}
		}

		file.Close()
	}
}

func makeCCYYMMDD(min, max time.Time) string {
	diffHrs := max.Sub(min).Hours()
	dt := min.Add(time.Duration(rand.Float64()*diffHrs) * time.Hour)
	return dt.Format("2006-01-02")
}

func makeProfile(hicn string, acoID string) record {
	dobMin, _ := time.Parse("2006-01-02", "1900-01-01")
	dobMax := time.Now().Add(-65 * 365 * 24 * time.Hour)

	p := record{
		hicn:       field{l: 11, v: hicn},                        // HICN
		blk:        field{l: 10, v: ""},                          // Beneficiary link key; blank permitted
		firstName:  field{l: 30, v: "First"},                     // Beneficiary first name
		middleName: field{l: 30, v: "Middle"},                    // Beneficiary middle name
		lastName:   field{l: 40, v: "Last"},                      // Beneficiary last name
		dob:        field{l: 8, v: makeCCYYMMDD(dobMin, dobMax)}, // Beneficiary date of birth; getCCYYMMDD
		addr1:      field{l: 55, v: "1 Main St."},                // Beneficiary address line 1
		addr2:      field{l: 55},                                 // Beneficiary address line 2
		addr3:      field{l: 55},                                 // Beneficiary address line 3
		city:       field{l: 40, v: "City"},                      // Beneficiary city
		state:      field{l: 2, v: "ST"},                         // Beneficiary state
		zip5:       field{l: 5, v: "00000"},                      // Beneficiary first five digits of ZIP code
		zip4:       field{l: 4, v: ""},                           // Beneficiary last four digits of ZIP code
		gender:     field{l: 1, v: "U"},                          // Beneficiary gender; M, F, U
		acoID:      field{l: 5, v: acoID},                        // ACO identifier
		acoName:    field{l: 70, v: ""},                          // ACO legal name
	}

	return p
}

func makeRecords(profile record) []record {
	records := []record{}
	// Create 0-5 records for this profile
	ct := rand.Intn(6)

	for i := 0; i < ct; i++ {
		r := profile
		r.encDate = field{l: 8}                                          // Encounter date
		r.effDate = field{l: 8, v: makeCCYYMMDD(time.Now(), time.Now())} // Beneficiary data sharing effective date
		r.srcCode = field{l: 5}                                          // Beneficiary data sharing source code
		r.mechCode = field{l: 1}                                         // Beneficiary option data sharing decision mechanism code
		r.prefInd = field{l: 1, v: makePrefInd()}                        // Beneficiary data sharing preference indicator
		r.saEffDate = field{l: 8}                                        // Beneficiary substance abuse data sharing effective date
		r.saSrcCode = field{l: 5}                                        // Beneficiary substance abuse data sharing source code
		r.saMechCode = field{l: 1}                                       // Beneficiary option substance abuse decision mechanism code
		r.saPrefInd = field{l: 1}                                        // Beneficiary substance abuse data sharing preference indicator

		records = append(records, r)
	}

	return records
}

func makePrefInd() string {
	i := []string{"Y", "N", ""}
	return i[rand.Intn(3)]
}
