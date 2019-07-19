package ssas

import (
	"fmt"
	"log"

	"github.com/jinzhu/gorm"
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
	GroupID string    `gorm:"unique;not null" json:"group_id"`
	Data    GroupData `gorm:"type:jsonb" json:"data"`
}

func CreateGroup(gd GroupData) (Group, error) {
	event := Event{Op: "CreateGroup", TrackingID: gd.ID}
	OperationStarted(event)

	if gd.ID == "" {
		err := fmt.Errorf("group_id cannot be blank")
		event.Help = err.Error()
		OperationFailed(event)
		return Group{}, err
	}

	g := Group{
		GroupID: gd.ID,
		Data:    gd,
	}

	db := GetGORMDbConnection()
	defer Close(db)
	err := db.Save(&g).Error
	if err != nil {
		event.Help = err.Error()
		OperationFailed(event)
		return Group{}, err
	}

	OperationSucceeded(event)
	return g, nil
}

func UpdateGroup(id string, gd GroupData) (Group, error) {
	event := Event{Op: "UpdateGroup", TrackingID: id}
	OperationStarted(event)

	g := Group{}
	db := GetGORMDbConnection()
	defer Close(db)
	if db.Where("id = ?", id).Find(&g).RecordNotFound() {
		err := fmt.Errorf("record not found for id=%s", id)
		event.Help = err.Error()
		OperationFailed(event)
		return Group{}, err
	}

	gd.ID = g.Data.ID
	gd.Name = g.Data.Name

	g.Data = gd
	err := db.Save(&g).Error
	if err != nil {
		event.Help = err.Error()
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
