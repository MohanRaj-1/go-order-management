package customer

import "context"

type Repository interface {
	Create(ctx context.Context, customer *Customer) error
	GetByID(ctx context.Context, id int64) (*Customer, error)
}
