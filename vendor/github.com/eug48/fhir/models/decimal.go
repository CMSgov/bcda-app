package models

import (
	"fmt"
	"github.com/eug48/fhir/utils"
)

type Decimal struct {
	From float64 `bson:"__from,omitempty"   json:"__from,omitempty"`
	To   float64 `bson:"__to,omitempty"     json:"__to,omitempty"`
	Num  float64 `bson:"__num,omitempty"    json:"__num,omitempty"`
	Str  string  `bson:"__strNum,omitempty" json:"__strNum,omitempty"`
}

func (d *Decimal) UnmarshalJSON(data []byte) (err error) {
	str := string(data)
	tmp, err := NewDecimal(str)
	if err == nil {
		*d = *tmp
	}
	return
}
func (f Decimal) MarshalJSON() ([]byte, error) {
	if f.Str == "" {
		return nil, fmt.Errorf("Decimal.MarshalJSON: empty string")
	}
	return []byte(f.Str), nil
}

func NewDecimal(str string) (*Decimal, error) {
	number := utils.ParseNumber(str)
	if number.Value == nil {
		return nil, fmt.Errorf("NewDecimal: failed to parse string (%s)", str)
	}

	num, _ := number.Value.Float64()
	numFrom, _ := number.RangeLowIncl().Float64()
	numTo, _ := number.RangeHighExcl().Float64()

	return &Decimal{
		Str:  str,
		Num:  num,
		From: numFrom,
		To:   numTo,
	}, nil
}
