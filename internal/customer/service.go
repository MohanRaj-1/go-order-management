package customer

import (
	"context"
	"fmt"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) CreateCustomer(
	ctx context.Context,
	name string,
	email string,
) (*Customer, error) {
	customer, err := NewCustomer(name, email)
	if err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, customer); err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}

	return customer, nil
}

func (s *Service) GetCustomerByID(
	ctx context.Context,
	id int64,
) (*Customer, error) {
	if id <= 0 {
		return nil, ErrInvalidCustomerID
	}

	customer, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get customer by ID: %w", err)
	}

	return customer, nil
}
