package public

import (
	"fmt"
	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/service"
	"github.com/dgrijalva/jwt-go"
	"time"
)

// MintMFAToken generates a tokenstring for MFA endpoints
func MintMFAToken(oktaID string) (*jwt.Token, string, error) {
	claims := service.CommonClaims{
		TokenType: "MFAToken",
		OktaID: oktaID,
	}

	if err := checkTokenClaims(&claims, claims.TokenType); err != nil {
		return nil, "", err
	}

	return server.MintToken(claims, time.Now().Unix(), time.Now().Add(server.TokenTTL).Unix())
}

// MintRegistrationToken generates a tokenstring for system self-registration endpoints
func MintRegistrationToken(oktaID string, groupIDs []string) (*jwt.Token, string, error) {
	claims := service.CommonClaims{
		TokenType: "RegistrationToken",
		OktaID: oktaID,
		GroupIDs: groupIDs,
	}

	if err := checkTokenClaims(&claims, claims.TokenType); err != nil {
		return nil, "", err
	}

	return server.MintTokenWithDuration(claims, server.TokenTTL)
}

// MintAccessToken generates a tokenstring that expires in server.TokenTTL time
func MintAccessToken(acoID string, data interface{}) (*jwt.Token, string, error) {
	claims := service.CommonClaims{
		TokenType: "AccessToken",
		ACOID: acoID,
		Data:  data,
	}

	if err := checkTokenClaims(&claims, claims.TokenType); err != nil {
		fmt.Println("error in checkTokenClaims: " + err.Error())   // TODO: remove
		return nil, "", err
	}

	return server.MintTokenWithDuration(claims, server.TokenTTL)
}

func empty(arr []string) bool {
	empty := true
	for _, item := range arr {
		if item != "" {
			empty = false
			break
		}
	}
	return empty
}

func tokenValidity(tokenString string, tokenType string) error {
	tknEvent := ssas.Event{Op: "tokenValidity"}
	ssas.OperationStarted(tknEvent)
	t, err := server.VerifyToken(tokenString)
	if err != nil {
		tknEvent.Help = err.Error()
		ssas.OperationFailed(tknEvent)
		return err
	}

	c := t.Claims.(*service.CommonClaims)

	err = checkAllClaims(c, tokenType)
	if err != nil {
		tknEvent.Help = err.Error()
		ssas.OperationFailed(tknEvent)
		return err
	}

	err = c.Valid()
	if err != nil {
		tknEvent.Help = err.Error()
		ssas.OperationFailed(tknEvent)
		return err
	}

	ssas.OperationSucceeded(tknEvent)
	return nil
}

func checkAllClaims(claims *service.CommonClaims, requiredTokenType string) error {
	if err := server.CheckRequiredClaims(claims, requiredTokenType); err != nil {
		return err
	}

	if err := checkTokenClaims(claims, requiredTokenType); err != nil {
		return err
	}
	return nil
}

func checkTokenClaims(claims *service.CommonClaims, requiredTokenType string) error {
	switch claims.TokenType {
	case "MFAToken":
		if claims.OktaID == "" {
			return fmt.Errorf("missing MFA claim(s)")
		}
	case "RegistrationToken":
		if empty(claims.GroupIDs) {
			return fmt.Errorf("missing registration claim(s)")
		}
	case "AccessToken":
		if claims.ACOID == "" {
			return fmt.Errorf("must have ACOID and data")
		}
	default:
		return fmt.Errorf("missing token type claim")
	}

	return nil
}