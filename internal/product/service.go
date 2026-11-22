package product

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MohanRaj-1/go-order-management/internal/database"
	"github.com/MohanRaj-1/go-order-management/internal/inventory"
)

type Service struct {
	productRepository  Repository
	transactionManager *database.TransactionManager
}

func NewService(
	db *sql.DB,
	transactionManager *database.TransactionManager,
) *Service {
	return &Service{
		productRepository:  NewPostgresRepository(db),
		transactionManager: transactionManager,
	}
}

func (s *Service) CreateProduct(
	ctx context.Context,
	sku string,
	name string,
	description string,
	price int64,
) (*Product, error) {
	product, err := NewProduct(sku, name, description, price)
	if err != nil {
		return nil, err
	}

	err = s.transactionManager.WithinTransaction(ctx, func(tx *sql.Tx) error {
		productRepository := NewPostgresRepository(tx)
		inventoryRepository := inventory.NewPostgresRepository(tx)

		if err := productRepository.Create(ctx, product); err != nil {
			return fmt.Errorf("create product: %w", err)
		}

		if err := inventoryRepository.Create(ctx, product.ID); err != nil {
			return fmt.Errorf("create initial inventory: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return product, nil
}

func (s *Service) GetProductByID(
	ctx context.Context,
	id int64,
) (*Product, error) {
	product, err := s.productRepository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get product by ID %d: %w", id, err)
	}

	return product, nil
}
