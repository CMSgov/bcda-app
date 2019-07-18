package ssas

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
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

	if gd.ID == "" {
		err := fmt.Errorf("group_id cannot be blank")
		event.Help = err.Error()
		OperationFailed(event)
		return Group{}, err
	}

	gdBytes, err := json.Marshal(gd)
	if err != nil {
		event.Help = err.Error()
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
		event.Help = err.Error()
		OperationFailed(event)
		return Group{}, err
	}

	OperationSucceeded(event)
	return g, nil
}

func UpdateGroup(id uint64, gd GroupData) (Group, error) {
	event := Event{Op: "UpdateGroup", TrackingID: string(id)}
	OperationStarted(event)

	g := Group{}
	db := GetGORMDbConnection()
	defer Close(db)
	if db.First(&g, id).RecordNotFound() {
		err := fmt.Errorf("record not found for id=%v", id)
		event.Help = err.Error()
		OperationFailed(event)
		return Group{}, err
	}

	oldGD := GroupData{}
	err := oldGD.Scan(&g.Data)
	if err != nil {
		event.Help = err.Error()
		OperationFailed(event)
		return Group{}, err
	}

	gd.ID = oldGD.ID
	gd.Name = oldGD.Name

	gdBytes, err := json.Marshal(gd)
	if err != nil {
		event.Help = err.Error()
		OperationFailed(event)
		return Group{}, err
	}

	g.Data = postgres.Jsonb{RawMessage: gdBytes}
	err = db.Save(&g).Error
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

// Make the GroupData struct implement the driver.Valuer interface. This method
// simply returns the JSON-encoded representation of the struct.
func (gd GroupData) Value() (driver.Value, error) {
	return json.Marshal(gd)
}

// Make the GroupData struct implement the sql.Scanner interface. This method
// simply decodes a JSON-encoded value into the struct fields.
func (gd *GroupData) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &gd)
}

type Resource struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}
