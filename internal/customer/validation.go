package customer

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

const (
	maxCustomerNameLength  = 150
	maxCustomerEmailLength = 254
)

func validateName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" || utf8.RuneCountInString(name) > maxCustomerNameLength {
		return ErrInvalidName
	}

	return nil
}

func validateEmail(email string) error {
	email = strings.TrimSpace(email)

	if email == "" || len(email) > maxCustomerEmailLength {
		return ErrInvalidEmail
	}

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return ErrInvalidEmail
	}

	return nil
}
