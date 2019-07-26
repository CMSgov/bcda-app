package public

import (
	"context"
	"fmt"
	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/service"
	"net/http"
	"regexp"

	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/responseutils"  // TODO: factor this requirement out
)


func readGroupID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			rd ssas.AuthRegData
			err error
		)
		if rd, err = readRegData(r); err != nil {
			rd = ssas.AuthRegData{}
		}

		if rd.GroupID = r.Header.Get("x-group-id"); rd.GroupID == "" {
			service.GetLogEntry(r).Println("missing header x-group-id; request will fail")
		}
		ctx := context.WithValue(r.Context(), "rd", rd)
		service.LogEntrySetField(r, "rd", rd)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Puts the decoded token and identity values into the request context. Decoded values have been
// verified to be tokens signed by our server and to have not expired. Additional authorization
// occurs in RequireTokenAuth(). Only auth code should look at the token claims; API code should
// rely on the values in AuthData. We do this to insulate API code from the differences among
// Provider tokens.
func parseToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		fmt.Println("Auth header:", authHeader)
		authRegexp := regexp.MustCompile(`^Bearer (\S+)$`)
		authSubmatches := authRegexp.FindStringSubmatch(authHeader)
		if len(authSubmatches) < 2 {
			log.Warn("Invalid Authorization header value")
			next.ServeHTTP(w, r)
			return
		}

		tokenString := authSubmatches[1]
		token, err := server.VerifyToken(tokenString)
		if err != nil {
			log.Errorf("Unable to decode Authorization header value; %s", err)
			next.ServeHTTP(w, r)
			return
		}

		var rd ssas.AuthRegData
		if rd, err = readRegData(r); err != nil {
			rd = ssas.AuthRegData{}
		}

		if claims, ok := token.Claims.(*service.CommonClaims); ok && token.Valid {
			rd.AllowedGroupIDs = claims.GroupIDs
			rd.OktaID = claims.OktaID
		}
		ctx := context.WithValue(r.Context(), "token", token)
		ctx = context.WithValue(ctx, "rd", rd)
		service.LogEntrySetField(r, "rd", rd)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requireRegTokenAuth(next http.Handler) http.Handler {
	return tokenAuth(next, "RegistrationToken")
}

func requireMFATokenAuth(next http.Handler) http.Handler {
	return tokenAuth(next, "MFAToken")
}

func tokenAuth(next http.Handler, tokenType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Context().Value("token")
		if token == nil {
			log.Error("No token found")
			respond(w, http.StatusUnauthorized)
			return
		}

		if token, ok := token.(*jwt.Token); ok {
			err := server.AuthorizeAccess(token.Raw, tokenType)
			if err != nil {
				log.Error(err)
				respond(w, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		}
	})
}

func respond(w http.ResponseWriter, status int) {
	oo := responseutils.CreateOpOutcome(responseutils.Error, responseutils.Exception, "", responseutils.TokenErr)
	responseutils.WriteError(oo, w, status)
}
