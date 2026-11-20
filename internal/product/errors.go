package product

import "errors"

var (
	ErrInvalidSKU   = errors.New("invalid product SKU")
	ErrInvalidName  = errors.New("invalid product name")
	ErrInvalidPrice = errors.New("invalid product price")
)
