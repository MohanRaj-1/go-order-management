package inventory

import "context"

type Repository interface {
	Create(ctx context.Context, productID int64) error

	GetByProductID(ctx context.Context, productID int64) (*Inventory, error)

	AdjustPhysicalQuantity(ctx context.Context, productID int64, delta int32) error

	Reserve(ctx context.Context, productID int64, quantity int32) error

	Release(ctx context.Context, productID int64, quantity int32) error
}
