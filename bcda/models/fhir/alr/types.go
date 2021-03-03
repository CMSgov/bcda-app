package alr

import (
	"time"

	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
)

func mbiIdentifier(mbi string) *fhirdatatypes.Identifier {
	return &fhirdatatypes.Identifier{
		System: fhirURI("http://hl7.org/fhir/sid/us-mbi"),
		Value:  fhirString(mbi),
	}
}

func fhirDate(t time.Time) *fhirdatatypes.Date {
	micros := t.UnixNano() / int64(time.Microsecond)
	return &fhirdatatypes.Date{ValueUs: micros, Precision: fhirdatatypes.Date_DAY}
}

func fhirDateTime(t time.Time) *fhirdatatypes.DateTime {
	micros := t.UnixNano() / int64(time.Microsecond)
	return &fhirdatatypes.DateTime{ValueUs: micros, Precision: fhirdatatypes.DateTime_DAY}
}

func fhirInstant(t time.Time) *fhirdatatypes.Instant {
	micros := t.UnixNano() / int64(time.Microsecond)
	return &fhirdatatypes.Instant{ValueUs: micros, Precision: fhirdatatypes.Instant_SECOND}
}

func fhirString(s string) *fhirdatatypes.String {
	return &fhirdatatypes.String{Value: s}
}

func fhirURI(uri string) *fhirdatatypes.Uri {
	return &fhirdatatypes.Uri{Value: uri}
}
