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

// MarshalJSON dual-writes Earliest/Latest and legacy LowerBound/UpperBound keys
// to maintain forward compatibility (if an older worker node processes a newly enqueued job).
// Remove once this release is fully deployed.
func (cw ClaimsWindow) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Earliest   time.Time `json:"Earliest,omitempty"`
		Latest     time.Time `json:"Latest,omitempty"`
		LowerBound time.Time `json:"LowerBound,omitempty"`
		UpperBound time.Time `json:"UpperBound,omitempty"`
	}{
		Earliest:   cw.Earliest,
		Latest:     cw.Latest,
		LowerBound: cw.Earliest,
		UpperBound: cw.Latest,
	})
}

// UnmarshalJSON ensures backward compatibility for jobs enqueued with legacy LowerBound/UpperBound keys.
// Remove once this release is fully deployed.
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
