package product

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/MohanRaj-1/go-order-management/internal/database"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresRepository struct {
	db database.DBTX
}

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(db database.DBTX) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, product *Product) error {

	if product == nil {
		return errors.New("create product: product cannot be nil")
	}

	const query = `
		INSERT INTO products (sku, name, description, price, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	// PostgreSQL numeric represents the price as paise multiplied by 10^-2.
	price := pgtype.Numeric{Int: big.NewInt(product.Price), Exp: -2, Valid: true}

	var id int64
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(
		ctx,
		query,
		product.SKU,
		product.Name,
		product.Description,
		price,
		product.Status,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("create product with SKU %q: %w", product.SKU, err)
	}

	product.ID = id
	product.CreatedAt = createdAt
	product.UpdatedAt = updatedAt
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (*Product, error) {
	// NUMERIC(10,2) has at most two fractional digits, so scaling by 100 is exact.
	const query = `
		SELECT id, sku, name, COALESCE(description, ''), (price * 100)::bigint,
		       status, created_at, updated_at
		FROM products
		WHERE id = $1
	`

	var product Product
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Status,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get product by ID %d: %w", id, ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get product by ID %d: %w", id, err)
	}
	return &product, nil
}
