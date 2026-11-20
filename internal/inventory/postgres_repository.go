package inventory

import (
	"context"
	"fmt"

	"github.com/MohanRaj-1/go-order-management/internal/database"
)

type PostgresRepository struct {
	db database.DBTX
}

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(db database.DBTX) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, productID int64) error {
	const query = `
		INSERT INTO inventory (product_id)
		VALUES ($1)
	`

	if _, err := r.db.ExecContext(ctx, query, productID); err != nil {
		return fmt.Errorf("create inventory for product ID %d: %w", productID, err)
	}

	return nil
}
