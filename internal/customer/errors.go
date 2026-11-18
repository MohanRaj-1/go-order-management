package customer

import "errors"

var (
	ErrNotFound          = errors.New("customer not found")
	ErrInvalidName       = errors.New("invalid customer name")
	ErrInvalidEmail      = errors.New("invalid customer email")
	ErrInvalidCustomerID = errors.New("invalid customer ID")
)
