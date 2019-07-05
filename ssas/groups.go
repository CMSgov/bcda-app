package ssas

import (
	"log"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
)

func InitializeGroupModels() *gorm.DB {
	log.Println("Initialize group models")
	db := GetGORMDbConnection()
	defer Close(db)

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