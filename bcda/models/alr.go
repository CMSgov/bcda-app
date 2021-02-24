package models

import (
	"time"
)

type Alr struct {
	ID            uint
	ACOID         uint // Link to AlrMetaData
	BeneMBI       string
	BeneHIC       string
	BeneFirstName string
	BeneLastName  string
	BeneSex       string
	BeneDOB       time.Time
	BeneDOD       time.Time
	Timestamp     time.Time
	KeyValue      string // This stores all the other "violate" fields
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AlrMetaData struct {
	MetaDataID uint // Foreign key
	ACO        string
	Timestamp  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Thought in progress...
/* type CSVContents struct {
	Dataframe dataframe.DataFrame
} */
