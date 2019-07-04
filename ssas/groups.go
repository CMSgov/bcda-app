package ssas

import (
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"log"
)

/*
	InitializeGroupModels will call gorm.DB.AutoMigrate() for Group{}
 */
func InitializeGroupModels() *gorm.DB {
	log.Println("Initialize group models")
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	db.AutoMigrate(
		&Group{},
	)

	return db
}

type Group struct {
	gorm.Model
	GroupID string         `gorm:"unique;not null" json:"group_id"`
	Data    postgres.Jsonb `json:"data"`
}

type GroupData struct {
	Name      string     `json:"name"`
	Users     []string   `json:"users"`
	Scopes    []string   `json:"scopes"`
	Systems   []System   `gorm:"foreignkey:GroupID;association_foreignkey:GroupID" json:"systems"`
	Resources []Resource `json:"resources"`
}

type Resource struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}