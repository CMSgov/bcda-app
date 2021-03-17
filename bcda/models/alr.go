package models

import (
	"time"
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

// Data Structure for Jobs
type AlrJobs struct {
	ID     uint
	CMSID  string
	Status JobStatus
}
type JobAlrEnqueueArgs struct {
	ID         uint
	CMSID      string
	MBIs       []string
	LowerBound time.Time
	UpperBound time.Time
}
