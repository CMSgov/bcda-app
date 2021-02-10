package postgres

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/models"
)

// Defining termination allows us to implement the Scanner and Valuer interface
// This allows us to read/write termination data from the postgres database
type termination struct {
	*models.Termination
}

// Value JSON encodes the termination data
func (t termination) Value() (driver.Value, error) {
	// Returning nil ensure that we leave the Termination column unset
	if t.Termination == nil {
		return nil, nil
	}
	return json.Marshal(t)
}

// Scan JSON decodes the provided src into the termination type.
// It handles NULL values.
func (t *termination) Scan(src interface{}) error {
	// Should be able to handle NULL value
	if src == nil {
		return nil
	}
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("could not convert %v to []byte", src)
	}
	return json.Unmarshal(b, &t.Termination)
}
