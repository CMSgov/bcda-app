package admin

import (
	"encoding/json"
	"fmt"
	"github.com/CMSgov/bcda-app/ssas"
	"net/http"
)

func createSystem(w http.ResponseWriter, r *http.Request) {
	type system struct {
		ClientName string `json:"client_name"`
		GroupID    string `json:"group_id"`
		Scope      string `json:"scope"`
		PublicKey  string `json:"public_key"`
		TrackingID string `json:"tracking_id"`
	}

	sys := system{}
	if err := json.NewDecoder(r.Body).Decode(&sys); err != nil {
		http.Error(w, "Could not create system due to invalid request body", http.StatusBadRequest)
		return
	}

	creds, err := ssas.RegisterSystem(sys.ClientName, sys.GroupID, sys.Scope, sys.PublicKey, sys.TrackingID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not create system. Error: %s", err), http.StatusBadRequest)
		return
	}

	credsJSON, err := json.Marshal(creds)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, string(credsJSON))
}