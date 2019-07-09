package admin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/CMSgov/bcda-app/ssas"
)

func createGroup(w http.ResponseWriter, r *http.Request) {
	gd := ssas.GroupData{}
	err := json.NewDecoder(r.Body).Decode(&gd)
	if err != nil {
		http.Error(w, "Failed to create group due to invalid request body", http.StatusBadRequest)
		return
	}

	g, err := ssas.CreateGroup(gd)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create group. Error: %s", err), http.StatusBadRequest)
		return
	}

	groupJSON, err := json.Marshal(g)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, string(groupJSON))
}
