package auth

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/dgrijalva/jwt-go"
)

type ClientID string
type TTL int

// TODO: check for pointer passing; test suite
func GetOpt(bucket interface{}, params ...interface{}) error {
	fmt.Println("Beginning of GetOpt")
	for _, p := range params {
		fmt.Printf(" Investigating param %v\n", p)
		if reflect.TypeOf(p) == reflect.ValueOf(bucket).Type() {
			switch t := p.(type) {
			case *ClientID:
				fmt.Println("  It's a *ClientID!")
				v := p.(*ClientID)
				fmt.Printf("    value %v\n", v)
				reflect.Indirect(reflect.ValueOf(bucket)).Set(reflect.ValueOf(*v))
				fmt.Println(bucket)
			case *TTL:
				fmt.Println("  It's a *TTL!")
				v := p.(*TTL)
				fmt.Printf("    value %v\n", v)
				reflect.Indirect(reflect.ValueOf(bucket)).Set(reflect.ValueOf(*v))
				fmt.Println(bucket)
			default:
				return fmt.Errorf("option type %v not implemented ", t)
			}

			return nil
		}
	}
	fmt.Println(" Returning with no match found")
	return errors.New("option type not found ")
}

// Provider is an interface for operations performed through an authentication provider.
type Provider interface {
	RegisterClient(clientID ClientID, params ...interface{}) (string, error)
	UpdateClient(clientID ClientID, params ...interface{}) ([]interface{}, error)
	DeleteClient(clientID ClientID, params ...interface{}) error

	GenerateClientCredentials(clientID ClientID, params ...interface{}) (interface{}, error)
	RevokeClientCredentials(clientID ClientID, params ...interface{}) error

	RequestAccessToken(clientID ClientID, params ...interface{}) (jwt.Token, error)
	RevokeAccessToken(tokenString string) error

	ValidateAccessToken(tokenString string) error
	DecodeAccessToken(tokenString string) (jwt.Token, error)
}
