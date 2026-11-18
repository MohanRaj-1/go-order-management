package customer

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
	return &PostgresRepository{
		db: db,
	}
}

func (r *PostgresRepository) Create(
	ctx context.Context,
	customer *Customer,
) error {
	const query = `
        INSERT INTO customers (name, email)
        VALUES ($1, $2)
        RETURNING id, name, email, created_at, updated_at
    `

	err := r.db.QueryRowContext(
		ctx,
		query,
		customer.Name,
		customer.Email,
	).Scan(
		&customer.ID,
		&customer.Name,
		&customer.Email,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create customer: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByID(
	ctx context.Context,
	id int64,
) (*Customer, error) {
	const query = `
		SELECT id, name, email, created_at, updated_at
		FROM customers
		WHERE id = $1
	`

	var customer Customer

	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&customer.ID,
		&customer.Name,
		&customer.Email,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("get customer by ID %d: %w", id, err)
	}

	return &customer, nil
}
