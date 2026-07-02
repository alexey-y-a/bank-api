package account

import "errors"

var (
	ErrAccountNotFound   = errors.New("account not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrForbidden         = errors.New("access denied: account does not belong to user")
	ErrInvalidAmount     = errors.New("invalid amount: must be greater than zero")
)
