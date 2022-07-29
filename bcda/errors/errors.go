package errors

import "fmt"

type EntityNotFoundError struct {
	Err   error
	CMSID string
}

func (e *EntityNotFoundError) Error() string {
	return fmt.Sprintf("no aco found for cmsID %s: %s", e.CMSID, e.Err)
}

// const (
// 	Data = "Data"
//  	Parsing = "Parsing"
//  	Configuration = "Configuration"
// )

type ValidationError struct {
	Err error
	Msg string
	//Label string //Data, Parse, Configuration, etc
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("Validation Error. Msg: %s, Err: %s", e.Msg, e.Err)
}

type UnexpectedStatusCodeError struct {
	Err        error
	StatusCode int //500, 401, etc
}

func (e *UnexpectedStatusCodeError) Error() string {
	return fmt.Sprintf("Unexpected Status Code %d: %s", e.StatusCode, e.Err)
}

//able to re-authenticate / re-request a new token ?
type AuthenticateError struct {
	Err error
}
