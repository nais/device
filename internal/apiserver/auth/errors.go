package auth

import (
	"fmt"
)

// JWT token parsing errors.
// The token library does not have any standardised error types,
// so we need one here to accurately represent this type of error.
type ParseTokenError struct {
	err error
}

var _ error = &ParseTokenError{}

func (t ParseTokenError) Error() string {
	return fmt.Sprintf("parse token: %s", t.err)
}
