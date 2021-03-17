package models

import (
	"time"

	"github.com/pborman/uuid"
)

// Data Structure for storing information into database
type Alr struct {
	ID            uint
	MetaKey       uint // Foreign Key
	BeneMBI       string
	BeneHIC       string
	BeneFirstName string
	BeneLastName  string
	BeneSex       string
	BeneDOB       time.Time
	BeneDOD       time.Time
	KeyValue      map[string]string // All "violate" fields
	Timestamp     time.Time         // NOT in the database, from AlrMetaData
}

type AlrMetaData struct {
	ID        uint // Primary Key
	CMSID     string
	Timestamp time.Time
}

// Data Structure for Jobs... this uses the same postgreSQL table as Job
type AlrJobs struct {
	ID              uint
	CMSID           uuid.UUID
	Status          JobStatus
	JobCount        int
	TransactionTime time.Time
	RequestedURL    string
}
type JobAlrEnqueueArgs struct {
	ID         uint
	CMSID      string
	MBIs       []string
	LowerBound time.Time
	UpperBound time.Time
}
