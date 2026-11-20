package inventory

import "context"

type Repository interface {
	Create(ctx context.Context, productID int64) error
}
