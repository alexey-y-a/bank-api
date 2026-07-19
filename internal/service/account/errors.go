package account

import "errors"

var (
	ErrAccountNotFound = errors.New("account not found")
	ErrForbidden       = errors.New("access denied: account does not belong to user")
)
