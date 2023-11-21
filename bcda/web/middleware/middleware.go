package middleware

import (
	"fmt"
	"net/http"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/log"
)

func ConnectionClose(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		next.ServeHTTP(w, r)
	})
}

func SecurityHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if servicemux.IsHTTPS(r) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			w.Header().Set("Cache-Control", "no-cache; no-store; must-revalidate; max-age=0")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}
		next.ServeHTTP(w, r)
	})
}

func ACOEnabled(cfg *service.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ad, ok := r.Context().Value(auth.AuthDataContextKey).(auth.AuthData)
			if !ok {
				panic("AuthData should be set before calling this handler")
			}

			version, err := getVersion(r.URL.Path)
			if err != nil {
				log.API.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			rw, err := getRespWriter(version)
			if err != nil {
				log.API.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if cfg.IsACODisabled(ad.CMSID) {
				log.API.Error(fmt.Sprintf("failed to complete request, CMSID %s is not enabled", ad.CMSID))
				rw.Exception(w, http.StatusUnauthorized, responseutils.InternalErr, "")
				return
			}
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
