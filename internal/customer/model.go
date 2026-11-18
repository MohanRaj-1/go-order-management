package customer

import (
	"strings"
	"time"
)

type Customer struct {
	ID        int64
	Name      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewCustomer(name, email string) (*Customer, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	if err := validateName(name); err != nil {
		return nil, err
	}

	if err := validateEmail(email); err != nil {
		return nil, err
	}

	return &Customer{
		Name:  name,
		Email: email,
	}, nil
}
