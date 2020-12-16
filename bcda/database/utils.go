package database

import (
	"runtime"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func Close(db *gorm.DB) {
	dbc, err := db.DB()
	if err != nil {
		log.Infof("failed to retrieve db connection: %v", err)
		return
	}
	if err := dbc.Close(); err != nil {
		_, file, line, _ := runtime.Caller(1)
		log.Infof("failed to close db connection at %s#%d because %s", file, line, err)
	}
}
