package product

import (
	"strings"
	"unicode/utf8"
)

const (
	maxProductSKULength  = 50
	maxProductNameLength = 150
	maxProductPrice      = int64(9_999_999_999)
)

func validateSKU(sku string) error {
	sku = strings.TrimSpace(sku)

	if sku == "" || utf8.RuneCountInString(sku) > maxProductSKULength {
		return ErrInvalidSKU
	}

	return nil
}

func validateName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" || utf8.RuneCountInString(name) > maxProductNameLength {
		return ErrInvalidName
	}

	return nil
}

func validatePrice(price int64) error {
	if price < 0 || price > maxProductPrice {
		return ErrInvalidPrice
	}

	return nil
}
