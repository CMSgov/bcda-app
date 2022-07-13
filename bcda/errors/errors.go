package errors

import "fmt"

type EntityNotFoundError struct {
	Err   error
	CMSID string
}

func (e *EntityNotFoundError) Error() string {
	return fmt.Sprintf("no aco found for cmsID %s: %s", e.CMSID, e.Err)
}
