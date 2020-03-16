// These structs exist for assisting in unmarshalling results from the Blue Button back end.
// They are not intended to be persisted to the BCDA database as they will only ever be used in
// memory when working with BlueButton results. I am adding as little as possible now with the
// expectation that more fields will be added as needed.

package models

import (
	"time"
)

type Patient struct {
	// uuid identifier of this request
	ID   string `json:"id"`
	Meta struct {
		LastUpdated time.Time
	}
	Entry []struct {
		Resource struct {
			Identifier []struct {
				System string `json:"system"`
				Value  string `json:"value"`
			} `json:"identifier"`
			ID string `json:"id"`
		} `json:"resource"`
	} `json:"entry"`
}
