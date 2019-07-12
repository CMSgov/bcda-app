package ssas

import (
	"encoding/json"
	"log"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
)

/*
	InitializeGroupModels will call gorm.DB.AutoMigrate() for Group{}
*/
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

func CreateGroup(gd GroupData) (Group, error) {
	event := Event{Op: "CreateGroup", TrackingID: gd.ID}
	OperationStarted(event)

	gdBytes, err := json.Marshal(gd)
	if err != nil {
		event.Help = err
		OperationFailed(event)
		return Group{}, err
	}

	g := Group{
		GroupID: gd.ID,
		Data:    postgres.Jsonb{RawMessage: gdBytes},
	}

	db := GetGORMDbConnection()
	defer Close(db)
	err = db.Save(&g).Error
	if err != nil {
		event.Help = err
		OperationFailed(event)
		return Group{}, err
	}

	OperationSucceeded(event)
	return g, nil
}

type GroupData struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Users     []string   `json:"users"`
	Scopes    []string   `json:"scopes"`
	System    System     `gorm:"foreignkey:GroupID;association_foreignkey:GroupID" json:"system"`
	Resources []Resource `json:"resources"`
}

type Resource struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}
