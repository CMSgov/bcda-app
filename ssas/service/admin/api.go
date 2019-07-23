package admin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/CMSgov/bcda-app/ssas"
	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
)

func createGroup(w http.ResponseWriter, r *http.Request) {
	gd := ssas.GroupData{}
	err := json.NewDecoder(r.Body).Decode(&gd)
	if err != nil {
		http.Error(w, "Failed to create group due to invalid request body", http.StatusBadRequest)
		return
	}

	ssas.OperationCalled(ssas.Event{Op: "CreateGroup", TrackingID: gd.ID, Help: "calling from admin.createGroup()"})
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
	_, err = w.Write(groupJSON)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

func updateGroup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	gd := ssas.GroupData{}
	err := json.NewDecoder(r.Body).Decode(&gd)
	if err != nil {
		http.Error(w, "Failed to update group due to invalid request body", http.StatusBadRequest)
		return
	}

	ssas.OperationCalled(ssas.Event{Op: "UpdateGroup", TrackingID: id, Help: "calling from admin.updateGroup()"})
	g, err := ssas.UpdateGroup(id, gd)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update group. Error: %s", err), http.StatusBadRequest)
		return
	}

	groupJSON, err := json.Marshal(g)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(groupJSON)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

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

	ssas.OperationCalled(ssas.Event{Op: "RegisterClient", TrackingID: sys.TrackingID, Help: "calling from admin.createSystem()"})
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
	_, err = w.Write(credsJSON)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

func resetCredentials(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemID")

	system, err := ssas.GetSystemByID(systemID)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	trackingID := uuid.NewRandom().String()
	ssas.OperationCalled(ssas.Event{Op: "ResetSecret", TrackingID: trackingID, Help: "calling from admin.resetCredentials()"})
	secret, err := system.ResetSecret(trackingID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{ "client_id": "%s", "client_secret": "%s" }`, systemID, secret)
}

func deactivateSystemCredentials(w http.ResponseWriter, r *http.Request) {
	systemID := chi.URLParam(r, "systemID")

	system, err := ssas.GetSystemByID(systemID)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	err = system.DeactivateSecrets()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
