package models

import (
	"github.com/eug48/fhir/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/mgo.v2/bson"
)

type Precision string

const (
	Date      = "date"
	YearMonth = "year-month"
	Year      = "year"
	Timestamp = "timestamp"
	Time      = "time"
)

type FHIRDateTime struct {
	Time      time.Time
	Precision Precision
}

func (f FHIRDateTime) GetBSON() (interface{}, error) {

	// if f.Precision == Timestamp {
		// return f.Time, nil
	// }

	bytesForm, err := f.MarshalJSON()
	stringForm := string(bytesForm[1:len(bytesForm)-1]) // remove JSON quotes
	if err != nil {
		return nil, errors.Wrap(err, "FHIRDateTime.GetBSON: MarshalJSON failed")
	}

	date, err := utils.ParseDate(stringForm)
	if err != nil {
		return nil, errors.Wrap(err, "FHIRDateTime.GetBSON: ParseDate failed")
	}

	doc := []bson.DocElem{
		bson.DocElem{Name: "__from", Value: date.RangeLowIncl()},
		bson.DocElem{Name: "__to", Value: date.RangeHighExcl()},
		bson.DocElem{Name: "__strDate", Value: stringForm},
	}
	return doc, nil

}

func (f *FHIRDateTime) SetBSON(raw bson.Raw) error {
	// fmt.Printf("FHIRDateTime.SetBSON: %+v %s\n", raw, string(raw.Data))
	if raw.Kind == 2 {
		// string - e.g. type instant
		err := f.UnmarshalJSON([]byte("\"" + string(raw.Data[4:len(raw.Data)-1]) + "\""))
		if err != nil {
			return errors.Wrap(err, "FHIRDateTime.SetBSON --> UnmarshalJSON failed")
		}
		return nil
	} else if raw.Kind == 3 {

		var doc []bson.DocElem
		err := raw.Unmarshal(&doc)
		if err != nil {
			return errors.Wrap(err, "FHIRDateTime.SetBSON --> Unmarshal failed")
		}
		for _, elt := range doc {
			if elt.Name == "__strDate" {
				strDate, ok := elt.Value.(string)
				if !ok {
					return errors.New("FHIRDateTime.SetBSON: __strDate is not a string")
				}
				err = f.UnmarshalJSON([]byte("\"" + strDate + "\""))
				if err != nil {
					return errors.Wrap(err, "FHIRDateTime.SetBSON --> UnmarshalJSON failed")
				}
				return nil
			}
		}
		return fmt.Errorf("FHIRDateTime.GetBSON: could not find __strDate")
	} else if raw.Kind == 9 {
		// UTC datetime (int64)
		err := raw.Unmarshal(&f.Time)
		if err != nil {
			return errors.Wrap(err, "FHIRDateTime.SetBSON --> Unmarshal of UTC timestamp failed")
		}
		f.Precision = Timestamp
		return nil
	} else {
		return fmt.Errorf("FHIRDateTime.GetBSON: could not parse BSON kind %d", raw.Kind)
	}
}

func (f *FHIRDateTime) UnmarshalJSON(data []byte) (err error) {
	strData := string(data)
	if len(data) <= 12 {
		f.Precision = Precision("date")
		f.Time, err = time.ParseInLocation("\"2006-01-02\"", strData, time.Local)
		if err != nil {
			f.Precision = Precision("year-month")
			f.Time, err = time.ParseInLocation("\"2006-01\"", strData, time.Local)
		}
		if err != nil {
			f.Precision = Precision("year")
			f.Time, err = time.ParseInLocation("\"2006\"", strData, time.Local)
		}
		if err != nil {
			// TODO: should move time into a separate type
			f.Precision = Precision("time")
			f.Time, err = time.ParseInLocation("\"15:04:05\"", strData, time.Local)
		}
		if err != nil {
			err = fmt.Errorf("unable to parse DateTime: %s", strData)
			f.Precision = ""
		}

	} else {
		f.Precision = Precision("timestamp")
		f.Time = time.Time{}
		err = f.Time.UnmarshalJSON(data)
	}
	return err
}

func (f FHIRDateTime) MarshalJSON() ([]byte, error) {
	if f.Precision == Timestamp {
		return json.Marshal(f.Time.Format(time.RFC3339))
	} else if f.Precision == YearMonth {
		return json.Marshal(f.Time.Format("2006-01"))
	} else if f.Precision == Year {
		return json.Marshal(f.Time.Format("2006"))
	} else if f.Precision == Time {
		return json.Marshal(f.Time.Format("15:04:05"))
	} else if f.Precision == Date {
		return json.Marshal(f.Time.Format("2006-01-02"))
	} else {
		return nil, fmt.Errorf("FHIRDateTime.MarshalJSON: unrecognised precision: %s", f.Precision)
	}
}
