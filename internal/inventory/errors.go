package inventory

import (
	"errors"
)

var (
	ErrInvalidProductID          = errors.New("invalid product ID")
	ErrNotFound                  = errors.New("inventory not found")
	ErrInvalidAdjustment         = errors.New("invalid inventory adjustment")
	ErrInsufficientPhysicalStock = errors.New("physical quantity cannot be reduced below reserved stock or zero")
	ErrInvalidReservation        = errors.New("invalid reservation quantity")
	ErrInsufficientStock         = errors.New("insufficient available stock")
	ErrInsufficientReservedStock = errors.New("insufficient reserved stock")
)
