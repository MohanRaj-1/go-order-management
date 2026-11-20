package product

import "context"

type Repository interface {
	Create(ctx context.Context, product *Product) error
}
