package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

type MbiChar struct {
	intVal  int
	charVal byte
	isInt   bool
}

func (d *MbiChar) next() bool {
	if d.isInt {
		if d.intVal == 9 {
			d.intVal = 0
			return true
		} else {
			d.intVal = d.intVal + 1
			return false
		}
	} else {
		switch d.charVal {
		case 'S':
			d.charVal = 'L'
			return false
		case 'L':
			d.charVal = 'O'
			return false
		case 'O':
			d.charVal = 'I'
			return false
		case 'I':
			d.charVal = 'B'
			return false
		case 'B':
			d.charVal = 'Z'
			return false
		case 'Z':
			d.charVal = 'Z'
			return true
		}
	}

	return false
}

type MockMBI = []*MbiChar

// 	n1  int
// 	n2  string
// 	n3  int
// 	n4  int
// 	n5  string
// 	n6  int
// 	n7  int
// 	n8  string
// 	n9  string
// 	n10 int
// 	n11 int
// }

func next(mbi MockMBI) {
	idx := 10

	for mbi[idx].next() {
		idx--
	}
}

func toString(mbi MockMBI) string {
	array := make([]string, 0, 11)
	for _, val := range mbi {
		if val.isInt {
			array = append(array, fmt.Sprintf("%v", val.intVal))
		} else {
			array = append(array, string(val.charVal))
		}
	}

	return strings.Join(array, "")
}

func main() {
	mbi := []*MbiChar{
		{isInt: true, intVal: 0},
		{charVal: 'S'},
		{isInt: true, intVal: 0},
		{isInt: true, intVal: 0},
		{charVal: 'S'},
		{isInt: true, intVal: 0},
		{isInt: true, intVal: 0},
		{charVal: 'S'},
		{charVal: 'S'},
		{isInt: true, intVal: 0},
		{isInt: true, intVal: 0},
	}

	now := time.Now()
	dateStr := fmt.Sprintf("D%s.T%s0", now.Format("060102"), now.Format("150405"))
	cclf0filePath := fmt.Sprintf("T.TEST567.ACO.ZC0Y24.%s", dateStr)
	cclf8filePath := fmt.Sprintf("T.TEST567.ACO.ZC8Y24.%s", dateStr)
	cclf9filePath := fmt.Sprintf("T.TEST567.ACO.ZC9Y24.%s", dateStr)
	f, err := os.OpenFile(cclf8filePath, os.O_WRONLY|os.O_CREATE, 0644)

	if err != nil {
		panic(err)
	}

	writer := bufio.NewWriter(f)

	numRecords := 1000000

	for i := 0; i < numRecords; i++ {
		filler := randSeq(549 - 11)
		writer.WriteString(toString(mbi))
		writer.WriteString(filler)
		writer.WriteString("\n")
		next(mbi)
	}

	writer.Flush()

	cclf0 := fmt.Sprintf(`File Number  |File Description    |Total Records Count |Record Length
CCLF1  |Part A Claims Header File                  |          6|  292
CCLF2  |Part A Claims Revenue Center Detail File   |          6|  179
CCLF3  |Part A Procedure Code File                 |          6|   94
CCLF4  |Part A Diagnosis Code File                 |          6|   92
CCLF5  |Part B Physicians File                     |          6|  363
CCLF6  |Part B DME File                            |          6|  227
CCLF7  |Part D File                                |          6|  195
CCLF8  |Beneficiary Demographics File              |%11d|  549
CCLF9  |BENE XREF File                             |          2|   55
CCLFA  |Part A BE and Demo Codes File              |          6|  101
CCLFB  |Part B BE and Demo Codes File              |          6|   93`, numRecords)

	f, err = os.OpenFile(cclf0filePath, os.O_WRONLY|os.O_CREATE, 0644)

	if err != nil {
		panic(err)
	}

	writer = bufio.NewWriter(f)
	writer.WriteString(cclf0)
	writer.Flush()

	f, err = os.OpenFile(cclf9filePath, os.O_WRONLY|os.O_CREATE, 0644)

	if err != nil {
		panic(err)
	}

	writer = bufio.NewWriter(f)

	numRecords = 1000000
	for i := 0; i < numRecords; i++ {
		filler := randSeq(549)
		writer.WriteString(filler)
		writer.WriteString("\n")
	}

	writer.Flush()
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
