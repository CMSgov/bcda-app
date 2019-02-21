package database

import (
	"runtime"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

func Close(db *gorm.DB) {
	if err := db.Close(); err != nil {
		_, file, line, _ := runtime.Caller(1)
		log.Infof("failed to close db connection at %s#%d because %s", file, line, err)
	}
}
