package errors

import "fmt"

type EntityNotFoundError struct {
	Err   error
	CMSID string
}

func (e *EntityNotFoundError) Error() string {
	return fmt.Sprintf("no aco found for cmsID %s: %s", e.CMSID, e.Err)
}

type RequestorDataError struct {
	Err error
	Msg string
}

func (e *RequestorDataError) Error() string {
	return fmt.Sprintf("Requestor Data Error encountered - %s. Err: %s", e.Msg, e.Err)
}

type InternalParsingError struct {
	Err error
	Msg string
}

func (e *InternalParsingError) Error() string {
	return fmt.Sprintf("Internal Parsing Error encountered - %s. Err: %s", e.Msg, e.Err)
}

type ConfigError struct {
	Err error
	Msg string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("Configuration Error encountered - %s. Err: %s", e.Msg, e.Err)
}

type RequestTimeoutError struct {
	Err error
	Msg string
}

func (e *RequestTimeoutError) Error() string {
	return fmt.Sprintf("Request Timeout Error encountered - %s. Err: %s", e.Msg, e.Err)
}

type UnexpectedSSASError struct {
	Err            error
	Msg            string
	SsasStatusCode int
}

func (e *UnexpectedSSASError) Error() string {
	return fmt.Sprintf("Unexpected SSAS Error encountered - %s. Status Code: %v, Err: %s", e.Msg, e.SsasStatusCode, e.Err)
}

type SSASErrorUnauthorized struct {
	Err            error
	Msg            string
	SsasStatusCode int
}

func (e *SSASErrorUnauthorized) Error() string {
	return fmt.Sprintf("Unexpected SSAS Error encountered - %s. Status Code: %v, Err: %s", e.Msg, e.SsasStatusCode, e.Err)
}

type SSASErrorBadRequest struct {
	Err            error
	Msg            string
	SsasStatusCode int
}

func (e *SSASErrorBadRequest) Error() string {
	return fmt.Sprintf("Unexpected SSAS Error encountered - %s. Status Code: %v, Err: %s", e.Msg, e.SsasStatusCode, e.Err)
}

type ExpiredTokenError struct {
	Err error
	Msg string
}

func (e *ExpiredTokenError) Error() string {
	return fmt.Sprintf("Expired Token Error encountered - %s. Err: %s", e.Msg, e.Err)
}
