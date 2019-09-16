package admin

import (
	"fmt"
	"net/http"

	"github.com/CMSgov/bcda-app/ssas"
)

func requireBasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID, secret, ok := r.BasicAuth()
		if !ok {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		system, err := ssas.GetSystemByClientID(clientID)
		if err != nil {
			jsonError(w, http.StatusText(http.StatusUnauthorized), "invalid client id")
			return
		}

		savedSecret, err := system.GetSecret()
		if err != nil || !ssas.Hash(savedSecret).IsHashOf(secret) {
			jsonError(w, http.StatusText(http.StatusUnauthorized), "invalid client secret")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func jsonError(w http.ResponseWriter, error string, description string) {
	ssas.Logger.Printf("%s; %s", description, error)
	w.WriteHeader(http.StatusBadRequest)
	body := []byte(fmt.Sprintf(`{"error":"%s","error_description":"%s"}`, error, description))
	_, err := w.Write(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
