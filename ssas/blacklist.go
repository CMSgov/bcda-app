package ssas

import (
"fmt"
	"github.com/pborman/uuid"
	"log"
	"time"

	"github.com/jinzhu/gorm"
)

//	InitializeBlacklistModels will call gorm.DB.AutoMigrate() for BlacklistEntries{}
func InitializeBlacklistModels() *gorm.DB {
	log.Println("Initialize cache models")
	db := GetGORMDbConnection()
	defer Close(db)

	db.AutoMigrate(
		&BlacklistEntry{},
	)

	return db
}

type BlacklistEntry struct {
	gorm.Model
	Key				string	`gorm:"not null" json:"key"`
	EntryDate		int64	`gorm:"not null" json:"entry_date"`
	CacheExpiration	int64	`gorm:"not null" json:"cache_expiration"`
}

func CreateBlacklistEntry(key string, entryDate time.Time, cacheExpiration time.Time) (entry BlacklistEntry, err error) {
	event := Event{Op: "CreateBlacklistEntry", TrackingID: key}
	OperationStarted(event)

	if key == "" {
		err = fmt.Errorf("key cannot be blank")
		event.Help = err.Error()
		OperationFailed(event)
		return
	}

	be := BlacklistEntry{
		Key: key,
		EntryDate: entryDate.Unix(),
		CacheExpiration: cacheExpiration.UnixNano(),
	}

	db := GetGORMDbConnection()
	defer Close(db)
	err = db.Save(&be).Error
	if err != nil {
		event.Help = err.Error()
		OperationFailed(event)
		return
	}

	OperationSucceeded(event)
	entry = be
	return
}

func GetUnexpiredBlacklistEntries() (entries []BlacklistEntry, err error) {
	trackingID := uuid.NewRandom().String()
	event := Event{Op: "GetBlacklistEntries", TrackingID: trackingID}
	OperationStarted(event)

	db := GetGORMDbConnection()
	defer Close(db)
	err = db.Order("entry_date, cache_expiration").Where("cache_expiration > ?", time.Now().UnixNano()).Find(&entries).Error
	if err != nil {
		event.Help = err.Error()
		OperationFailed(event)
		return
	}

	OperationSucceeded(event)
	return
}