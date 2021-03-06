package models

import (
	"time"
)

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
	ACO       string
	Timestamp time.Time
}
