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
	Timestamp     time.Time
	KeyValue      map[string]string // All "violate" fields
}

type AlrMetaData struct {
	ID        uint // Primary Key
	ACO       string
	Timestamp time.Time
}
