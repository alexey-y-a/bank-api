package transfer

import "errors"

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrSameAccount       = errors.New("cannot transfer to the same account")
	ErrAccountNotFound   = errors.New("account not found")
	ErrForbidden         = errors.New("access denied: account does not belong to user")
)
