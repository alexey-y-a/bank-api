package card

import "errors"

var (
	ErrCardNotFound = errors.New("card not found")
	ErrInvalidCVV   = errors.New("invalid CVV code")
)
