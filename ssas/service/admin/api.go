package admin

import (
	"encoding/json"
	"fmt"
	"github.com/CMSgov/bcda-app/ssas"
	"net/http"
)

func createSystem(w http.ResponseWriter, r *http.Request) {
	type system struct {
		GroupID    string `json:"group_id"`
		ClientID   string `json:"client_id"`
		ClientName string `json:"client_name"`
		Scope      string `json:"scope"`
		PublicKey  string `json:"public_key"`
	}

	sys := system{}
	if err := json.NewDecoder(r.Body).Decode(&sys); err != nil {
		http.Error(w, "Could not create system due to invalid request body", http.StatusBadRequest)
		return
	}

	creds, err := ssas.RegisterSystem(sys.ClientName, sys.GroupID, sys.Scope, sys.PublicKey, sys.ClientID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not create system. Error: %s", err), http.StatusInternalServerError)
		return
	}

	credsJSON, err := json.Marshal(creds)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not create system. Error: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, string(credsJSON))
}