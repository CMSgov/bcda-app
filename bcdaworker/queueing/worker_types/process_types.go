package worker_types

import (
	"encoding/json"
	"time"

	"github.com/CMSgov/bcda-app/bcda/fhir"
)

const QUE_PROCESS_JOB = "ProcessJob"

type ClaimsWindow struct {
	Earliest time.Time `json:"Earliest,omitempty"`
	Latest   time.Time `json:"Latest,omitempty"`
}

// UnmarshalJSON ensures backward compatibility for jobs enqueued with legacy LowerBound/UpperBound keys.
func (cw *ClaimsWindow) UnmarshalJSON(data []byte) error {
	type Alias ClaimsWindow
	aux := struct {
		*Alias
		LowerBound time.Time `json:"LowerBound"`
		UpperBound time.Time `json:"UpperBound"`
	}{
		Alias: (*Alias)(cw),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if cw.Earliest.IsZero() && !aux.LowerBound.IsZero() {
		cw.Earliest = aux.LowerBound
	}
	if cw.Latest.IsZero() && !aux.UpperBound.IsZero() {
		cw.Latest = aux.UpperBound
	}
	return nil
}

type JobEnqueueArgs struct {
	ID              int // parent Job ID
	ACOID           string
	CMSID           string
	BeneficiaryIDs  []string
	ResourceType    string
	Since           string
	TypeFilter      fhir.TypeFilterSubquery
	TransactionID   string
	TransactionTime time.Time
	BBBasePath      string
	ClaimsWindow    ClaimsWindow
	DataType        string
}

// Needed by River (queue library)
func (jobargs JobEnqueueArgs) Kind() string {
	return QUE_PROCESS_JOB
}
