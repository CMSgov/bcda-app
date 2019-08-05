package ssas

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
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

func ListGroups(trackingID string) ([]Group, error) {
	event := Event{Op: "ListGroups", TrackingID: trackingID}
	OperationStarted(event)

	groups := []Group{}
	db := GetGORMDbConnection()
	defer Close(db)
	err := db.Find(&groups).Error
	if err != nil {
		event.Help = err.Error()
		OperationFailed(event)
		return []Group{}, err
	}

	OperationSucceeded(event)
	return groups, nil
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

func DeleteGroup(id string) error {
	event := Event{Op: "DeleteGroup", TrackingID: id}
	OperationStarted(event)

	db := GetGORMDbConnection()
	defer Close(db)
	g := Group{}
	err := db.Where("id = ?", id).First(&g).Error
	if err != nil {
		event.Help = err.Error()
		OperationFailed(event)
		return err
	}

	err = cascadeDeleteGroup(g)
	if err != nil {
		event.Help = err.Error()
		OperationFailed(event)
		return err
	}

	OperationSucceeded(event)
	return nil
}

// GetAuthorizedGroupsForOktaID returns a slice of GroupID's representing all groups this Okta user has rights to manage
// TODO: this is the slowest and most memory intensive way possible to implement this.  Refactor!
func GetAuthorizedGroupsForOktaID(oktaID string) ([]string, error) {
	db := GetGORMDbConnection()
	defer Close(db)

	var (
		result []string
	)

	groups := []Group{}
	err := db.Select("*").Find(&groups).Error
	if err != nil {
		return result, err
	}

	for _, group := range groups {
		for _, user := range group.Data.Users {
			if user == oktaID {
				result = append(result, group.GroupID)
			}
		}
	}

	return result, nil
}

func cascadeDeleteGroup(group Group) error {
	var (
		system        System
		encryptionKey EncryptionKey
		secret        Secret
		systemIds     []int
		db            = GetGORMDbConnection()
	)
	defer Close(db)

	err := db.Table("systems").Where("group_id = ?", group.GroupID).Pluck("ID", &systemIds).Error
	if err != nil {
		return fmt.Errorf("unable to find associated systems: %s", err.Error())
	}

	err = db.Where("system_id IN (?)", systemIds).Delete(&encryptionKey).Error
	if err != nil {
		return fmt.Errorf("unable to delete encryption keys: %s", err.Error())
	}

	err = db.Where("system_id IN (?)", systemIds).Delete(&secret).Error
	if err != nil {
		return fmt.Errorf("unable to delete secrets: %s", err.Error())
	}

	err = db.Where("id IN (?)", systemIds).Delete(&system).Error
	if err != nil {
		return fmt.Errorf("unable to delete systems: %s", err.Error())
	}

	err = db.Delete(&group).Error
	if err != nil {
		return fmt.Errorf("unable to delete group: %s", err.Error())
	}

	return nil
}

type GroupData struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Users     []string    `json:"users"`
	Scopes    []string    `json:"scopes"`
	System    System      `gorm:"foreignkey:GroupID;association_foreignkey:GroupID" json:"system"`
	Resources []Resource  `json:"resources"`
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

func FindByGroupID(groupID string) (Group, error){
	db := GetGORMDbConnection()
	defer Close(db)

	group := Group{GroupID:groupID}
	err := db.Find(&group).Error
	if err != nil {
		return group, err
	}

	return group, nil
}