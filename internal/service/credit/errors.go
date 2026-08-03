package credit

import "errors"

var (
	ErrCreditNotFound    = errors.New("credit not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrForbidden         = errors.New("access denied: credit does not belong to user")
)
