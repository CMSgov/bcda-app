package ssas

import (
"fmt"
	"github.com/pborman/uuid"
	"log"
	"time"

	"github.com/jinzhu/gorm"
)

//	InitializeCacheModels will call gorm.DB.AutoMigrate() for CacheEntries{}
func InitializeCacheModels() *gorm.DB {
	log.Println("Initialize cache models")
	db := GetGORMDbConnection()
	defer Close(db)

	db.AutoMigrate(
		&CacheEntry{},
	)

	return db
}

type CacheEntry struct {
	gorm.Model
	Key				string	`gorm:"not null" json:"key"`
	EntryDate		int64	`gorm:"not null" json:"entry_date"`
	CacheExpiration	int64	`gorm:"not null" json:"cache_expiration"`
}

func CreateCacheEntry(key string, entryDate time.Time, cacheExpiration time.Time) (entry CacheEntry, err error) {
	event := Event{Op: "CreateCacheEntry", TrackingID: key}
	OperationStarted(event)

	if key == "" {
		err = fmt.Errorf("key cannot be blank")
		event.Help = err.Error()
		OperationFailed(event)
		return
	}

	ce := CacheEntry{
		Key: key,
		EntryDate: entryDate.Unix(),
		CacheExpiration: cacheExpiration.UnixNano(),
	}

	db := GetGORMDbConnection()
	defer Close(db)
	err = db.Save(&ce).Error
	if err != nil {
		event.Help = err.Error()
		OperationFailed(event)
		return
	}

	OperationSucceeded(event)
	entry = ce
	return
}

func GetUnexpiredCacheEntries() (entries []CacheEntry, err error) {
	trackingID := uuid.NewRandom().String()
	event := Event{Op: "GetCacheEntries", TrackingID: trackingID}
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