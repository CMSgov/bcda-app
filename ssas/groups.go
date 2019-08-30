package ssas

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/jinzhu/gorm"
)

// InitializeGroupModels creates and updates the schema for groups
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
	XData   string    `gorm:"type:text" json:"xdata"`
	Data    GroupData `gorm:"type:jsonb" json:"data"`
}

func CreateGroup(gd GroupData) (Group, error) {
	event := Event{Op: "CreateGroup", TrackingID: gd.GroupID}
	OperationStarted(event)

	if gd.GroupID == "" {
		err := fmt.Errorf("group_id cannot be blank")
		event.Help = err.Error()
		OperationFailed(event)
		return Group{}, err
	}

	xd := gd.XData
	if xd != "" {
		if s, err := strconv.Unquote(xd); err == nil {
			xd = s
		}
	}
	g := Group{
		GroupID: gd.GroupID,
		XData:   xd,
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

	gd.GroupID = g.Data.GroupID
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
	GroupID   string     `json:"group_id"`
	Name      string     `json:"name"`
	XData     string     `json:"xdata"`
	Users     []string   `json:"users,omitempty"`
	Scopes    []string   `json:"scopes,omitempty"`
	Systems   []System   `gorm:"-" json:"systems,omitempty"`
	Resources []Resource `json:"resources,omitempty"`
}

// Value implements the driver.Value interface for GroupData.
func (gd GroupData) Value() (driver.Value, error) {
	systems, _ := GetSystemsByGroupID(gd.GroupID)

	gd.Systems = systems

	return json.Marshal(gd)
}

// Make the GroupData struct implement the sql.Scanner interface
func (gd *GroupData) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	if err := json.Unmarshal(b, &gd); err != nil {
		return err
	}
	systems, _ := GetSystemsByGroupID(gd.GroupID)

	gd.Systems = systems

	return nil
}

type Resource struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

func FindByGroupID(groupID string) (Group, error) {
	var (
		db    = GetGORMDbConnection()
		group Group
		err   error
	)
	defer Close(db)

	if db.Find(&group, "group_id = ?", groupID).RecordNotFound() {
		err = fmt.Errorf("no Group record found for groupID %s", groupID)
	}

	return group, err
}
