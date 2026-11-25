package inventory

import (
	"context"
	"database/sql"
	"errors"
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

func (r *PostgresRepository) GetByProductID(ctx context.Context, productID int64) (*Inventory, error) {
	const query = `
		SELECT id, product_id, physical_quantity, reserved_quantity, created_at, updated_at
		FROM inventory
		WHERE product_id = $1
	`

	var inventory Inventory
	err := r.db.QueryRowContext(ctx, query, productID).Scan(
		&inventory.ID,
		&inventory.ProductID,
		&inventory.PhysicalQuantity,
		&inventory.ReservedQuantity,
		&inventory.CreatedAt,
		&inventory.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get inventory for product ID %d: %w", productID, err)
	}
	return &inventory, nil
}

func (r *PostgresRepository) AdjustPhysicalQuantity(
	ctx context.Context,
	productID int64,
	delta int32,
) error {
	if delta == 0 {
		return ErrInvalidAdjustment
	}

	const query = `
        UPDATE inventory
        SET physical_quantity = physical_quantity::bigint + $1::bigint,
            updated_at = CURRENT_TIMESTAMP
        WHERE product_id = $2
          AND physical_quantity::bigint + $1::bigint >= reserved_quantity
          AND physical_quantity::bigint + $1::bigint BETWEEN 0 AND 2147483647
    `

	result, err := r.db.ExecContext(ctx, query, int64(delta), productID)
	if err != nil {
		return fmt.Errorf(
			"adjust physical quantity for product ID %d: %w",
			productID,
			err,
		)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"count rows affected by physical quantity adjustment for product ID %d: %w",
			productID,
			err,
		)
	}

	if rows > 0 {
		return nil
	}

	current, err := r.GetByProductID(ctx, productID)
	if err != nil {
		return err
	}

	newQuantity := int64(current.PhysicalQuantity) + int64(delta)
	if newQuantity < 0 || newQuantity < int64(current.ReservedQuantity) {
		return ErrInsufficientPhysicalStock
	}

	return ErrInvalidAdjustment
}

func (r *PostgresRepository) Reserve(ctx context.Context, productID int64, quantity int32) error {
	if quantity <= 0 {
		return ErrInvalidReservation
	}

	const query = `
		UPDATE inventory
		SET reserved_quantity = (reserved_quantity::bigint + $1::bigint)::integer,
		    updated_at = CURRENT_TIMESTAMP
		WHERE product_id = $2
		  AND physical_quantity::bigint - reserved_quantity::bigint >= $1::bigint
		  AND reserved_quantity::bigint + $1::bigint <= 2147483647
	`

	result, err := r.db.ExecContext(ctx, query, int64(quantity), productID)
	if err != nil {
		return fmt.Errorf("reserve inventory for product ID %d: %w", productID, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count rows affected by reservation for product ID %d: %w", productID, err)
	}
	if rows > 0 {
		return nil
	}

	if _, err := r.GetByProductID(ctx, productID); err != nil {
		return err
	}

	return ErrInsufficientStock
}

func (r *PostgresRepository) Release(ctx context.Context, productID int64, quantity int32) error {
	if quantity <= 0 {
		return ErrInvalidReservation
	}

	const query = `
		UPDATE inventory
		SET reserved_quantity = (reserved_quantity::bigint - $1::bigint)::integer,
		    updated_at = CURRENT_TIMESTAMP
		WHERE product_id = $2
		  AND reserved_quantity::bigint >= $1::bigint
	`

	result, err := r.db.ExecContext(ctx, query, int64(quantity), productID)
	if err != nil {
		return fmt.Errorf("release inventory for product ID %d: %w", productID, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count rows affected by release for product ID %d: %w", productID, err)
	}
	if rows > 0 {
		return nil
	}
	if _, err := r.GetByProductID(ctx, productID); err != nil {
		return err
	}

	return ErrInsufficientReservedStock
}
