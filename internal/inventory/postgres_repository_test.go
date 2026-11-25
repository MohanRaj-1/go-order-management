package inventory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/MohanRaj-1/go-order-management/internal/database"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Open(ctx, databaseURL, database.DefaultConfig())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	return db
}

// createTestProduct inserts a product inside the supplied transaction.
func createTestProduct(t *testing.T, ctx context.Context, tx *sql.Tx) int64 {
	t.Helper()

	sku := fmt.Sprintf("inventory-test-%d", time.Now().UnixNano())

	const query = `
		INSERT INTO products (sku, name, description, price, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var productID int64
	err := tx.QueryRowContext(
		ctx,
		query,
		sku,
		"Inventory Test Product",
		"Product created for inventory repository tests",
		100.00,
		"ACTIVE",
	).Scan(&productID)
	if err != nil {
		t.Fatalf("create test product: %v", err)
	}

	return productID
}

func TestPostgresRepository_Create(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	got, err := repository.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got == nil {
		t.Fatal("expected inventory, got nil")
	}

	if got.ProductID != productID {
		t.Errorf("ProductID = %d, want %d", got.ProductID, productID)
	}

	if got.PhysicalQuantity != 0 {
		t.Errorf("PhysicalQuantity = %d, want 0", got.PhysicalQuantity)
	}

	if got.ReservedQuantity != 0 {
		t.Errorf("ReservedQuantity = %d, want 0", got.ReservedQuantity)
	}

	if got.ID <= 0 {
		t.Errorf("expected generated inventory ID, got %d", got.ID)
	}

	if got.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be populated")
	}

	if got.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be populated")
	}
}

func TestPostgresRepository_GetByProductID(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	got, err := repository.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory by product ID: %v", err)
	}

	if got == nil {
		t.Fatal("expected inventory, got nil")
	}

	if got.ProductID != productID {
		t.Errorf("ProductID = %d, want %d", got.ProductID, productID)
	}

	if got.PhysicalQuantity != 0 {
		t.Errorf("PhysicalQuantity = %d, want 0", got.PhysicalQuantity)
	}

	if got.ReservedQuantity != 0 {
		t.Errorf("ReservedQuantity = %d, want 0", got.ReservedQuantity)
	}

	if got.ID <= 0 {
		t.Errorf("expected generated inventory ID, got %d", got.ID)
	}

	if got.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be populated")
	}

	if got.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be populated")
	}
}

func TestPostgresRepository_GetByProductID_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)

	got, err := repository.GetByProductID(ctx, -1)
	if got != nil {
		t.Errorf("expected nil inventory, got %+v", got)
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want %v", err, ErrNotFound)
	}
}

func TestPostgresRepository_AdjustPhysicalQuantity_Increase(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	if err := repository.AdjustPhysicalQuantity(ctx, productID, 10); err != nil {
		t.Fatalf("increase physical quantity: %v", err)
	}

	got, err := repository.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != 10 {
		t.Errorf("PhysicalQuantity = %d, want 10", got.PhysicalQuantity)
	}
}

func TestPostgresRepository_AdjustPhysicalQuantity_Decrease(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	if err := repository.AdjustPhysicalQuantity(ctx, productID, 10); err != nil {
		t.Fatalf("initial stock adjustment: %v", err)
	}

	if err := repository.AdjustPhysicalQuantity(ctx, productID, -4); err != nil {
		t.Fatalf("decrease physical quantity: %v", err)
	}

	got, err := repository.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != 6 {
		t.Errorf("PhysicalQuantity = %d, want 6", got.PhysicalQuantity)
	}
}

func TestPostgresRepository_AdjustPhysicalQuantity_ZeroDelta(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	err = repository.AdjustPhysicalQuantity(ctx, productID, 0)
	if !errors.Is(err, ErrInvalidAdjustment) {
		t.Errorf("error = %v, want %v", err, ErrInvalidAdjustment)
	}
}

func TestPostgresRepository_AdjustPhysicalQuantity_BelowZero(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	err = repository.AdjustPhysicalQuantity(ctx, productID, -1)
	if !errors.Is(err, ErrInsufficientPhysicalStock) {
		t.Errorf("error = %v, want %v", err, ErrInsufficientPhysicalStock)
	}

	got, err := repository.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != 0 {
		t.Errorf("PhysicalQuantity = %d, want 0", got.PhysicalQuantity)
	}
}

func TestPostgresRepository_AdjustPhysicalQuantity_BelowReserved(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	if err := repository.AdjustPhysicalQuantity(ctx, productID, 10); err != nil {
		t.Fatalf("add physical stock: %v", err)
	}

	if err := repository.Reserve(ctx, productID, 6); err != nil {
		t.Fatalf("reserve stock: %v", err)
	}

	err = repository.AdjustPhysicalQuantity(ctx, productID, -5)
	if !errors.Is(err, ErrInsufficientPhysicalStock) {
		t.Fatalf("error = %v, want %v", err, ErrInsufficientPhysicalStock)
	}

	got, err := repository.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != 10 {
		t.Errorf("PhysicalQuantity = %d, want 10", got.PhysicalQuantity)
	}

	if got.ReservedQuantity != 6 {
		t.Errorf("ReservedQuantity = %d, want 6", got.ReservedQuantity)
	}
}

func TestPostgresRepository_AdjustPhysicalQuantity_ToReservedQuantity(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	if err := repository.AdjustPhysicalQuantity(ctx, productID, 10); err != nil {
		t.Fatalf("add physical stock: %v", err)
	}

	if err := repository.Reserve(ctx, productID, 6); err != nil {
		t.Fatalf("reserve stock: %v", err)
	}

	if err := repository.AdjustPhysicalQuantity(ctx, productID, -4); err != nil {
		t.Fatalf("adjust physical quantity to reserved quantity: %v", err)
	}

	got, err := repository.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != 6 {
		t.Errorf("PhysicalQuantity = %d, want 6", got.PhysicalQuantity)
	}

	if got.ReservedQuantity != 6 {
		t.Errorf("ReservedQuantity = %d, want 6", got.ReservedQuantity)
	}

	if got.AvailableQuantity() != 0 {
		t.Errorf("AvailableQuantity() = %d, want 0", got.AvailableQuantity())
	}
}

func TestPostgresRepository_AdjustPhysicalQuantity_ExceedsMaxInteger(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	const maxInt32 = int32(2147483647)

	err = repository.AdjustPhysicalQuantity(ctx, productID, maxInt32)
	if err != nil {
		t.Fatalf("set physical quantity to maximum INTEGER: %v", err)
	}

	err = repository.AdjustPhysicalQuantity(ctx, productID, 1)
	if !errors.Is(err, ErrInvalidAdjustment) {
		t.Errorf("error = %v, want %v", err, ErrInvalidAdjustment)
	}

	got, err := repository.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != maxInt32 {
		t.Errorf("PhysicalQuantity = %d, want %d", got.PhysicalQuantity, maxInt32)
	}
}

func TestPostgresRepository_AdjustPhysicalQuantity_MaxInteger(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	const maxInt32 = int32(2147483647)

	if err := repository.AdjustPhysicalQuantity(ctx, productID, maxInt32); err != nil {
		t.Fatalf("set physical quantity to maximum INTEGER: %v", err)
	}

	got, err := repository.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != maxInt32 {
		t.Errorf("PhysicalQuantity = %d, want %d", got.PhysicalQuantity, maxInt32)
	}
}

func TestPostgresRepository_Reserve_Success(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	if err := repository.AdjustPhysicalQuantity(ctx, productID, 10); err != nil {
		t.Fatalf("add physical stock: %v", err)
	}

	if err := repository.Reserve(ctx, productID, 4); err != nil {
		t.Fatalf("reserve stock: %v", err)
	}

	got, err := repository.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != 10 {
		t.Errorf("PhysicalQuantity = %d, want 10", got.PhysicalQuantity)
	}

	if got.ReservedQuantity != 4 {
		t.Errorf("ReservedQuantity = %d, want 4", got.ReservedQuantity)
	}

	if got.AvailableQuantity() != 6 {
		t.Errorf("AvailableQuantity() = %d, want 6", got.AvailableQuantity())
	}
}

func TestPostgresRepository_Reserve_InsufficientStock(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	if err := repository.AdjustPhysicalQuantity(ctx, productID, 10); err != nil {
		t.Fatalf("add physical stock: %v", err)
	}

	if err := repository.Reserve(ctx, productID, 6); err != nil {
		t.Fatalf("reserve initial stock: %v", err)
	}

	err = repository.Reserve(ctx, productID, 5)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("error = %v, want %v", err, ErrInsufficientStock)
	}

	got, err := repository.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != 10 {
		t.Errorf("PhysicalQuantity = %d, want 10", got.PhysicalQuantity)
	}

	if got.ReservedQuantity != 6 {
		t.Errorf("ReservedQuantity = %d, want 6", got.ReservedQuantity)
	}
}

func TestPostgresRepository_Reserve_ZeroQuantity(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	err = repository.Reserve(ctx, productID, 0)
	if !errors.Is(err, ErrInvalidReservation) {
		t.Errorf("error = %v, want %v", err, ErrInvalidReservation)
	}
}

func TestPostgresRepository_Reserve_NegativeQuantity(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repository := NewPostgresRepository(tx)
	productID := createTestProduct(t, ctx, tx)

	if err := repository.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	err = repository.Reserve(ctx, productID, -5)
	if !errors.Is(err, ErrInvalidReservation) {
		t.Errorf("error = %v, want %v", err, ErrInvalidReservation)
	}
}

func TestPostgresRepository_Reserve_ExactAvailableStock(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	productID := createTestProduct(t, ctx, tx)
	repo := NewPostgresRepository(tx)

	if err := repo.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	if err := repo.AdjustPhysicalQuantity(ctx, productID, 10); err != nil {
		t.Fatalf("adjust physical quantity: %v", err)
	}

	if err := repo.Reserve(ctx, productID, 6); err != nil {
		t.Fatalf("reserve first quantity: %v", err)
	}

	if err := repo.Reserve(ctx, productID, 4); err != nil {
		t.Fatalf("reserve remaining quantity: %v", err)
	}

	got, err := repo.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != 10 {
		t.Errorf("physical quantity = %d, want 10", got.PhysicalQuantity)
	}

	if got.ReservedQuantity != 10 {
		t.Errorf("reserved quantity = %d, want 10", got.ReservedQuantity)
	}

	if got.AvailableQuantity() != 0 {
		t.Errorf("available quantity = %d, want 0", got.AvailableQuantity())
	}
}

func TestPostgresRepository_Reserve_MaxInteger(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	productID := createTestProduct(t, ctx, tx)
	repo := NewPostgresRepository(tx)

	const maxInt32 int32 = 2147483647

	if err := repo.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	if err := repo.AdjustPhysicalQuantity(ctx, productID, maxInt32); err != nil {
		t.Fatalf("adjust physical quantity: %v", err)
	}

	if err := repo.Reserve(ctx, productID, maxInt32); err != nil {
		t.Fatalf("reserve maximum quantity: %v", err)
	}

	if err := repo.Reserve(ctx, productID, 1); !errors.Is(err, ErrInsufficientStock) {
		t.Errorf("reserve additional quantity error = %v, want %v", err, ErrInsufficientStock)
	}

	got, err := repo.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != maxInt32 {
		t.Errorf("physical quantity = %d, want %d", got.PhysicalQuantity, maxInt32)
	}

	if got.ReservedQuantity != maxInt32 {
		t.Errorf("reserved quantity = %d, want %d", got.ReservedQuantity, maxInt32)
	}

	if got.AvailableQuantity() != 0 {
		t.Errorf("available quantity = %d, want 0", got.AvailableQuantity())
	}
}

func TestPostgresRepository_Release_Success(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	productID := createTestProduct(t, ctx, tx)
	repo := NewPostgresRepository(tx)

	if err := repo.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	if err := repo.AdjustPhysicalQuantity(ctx, productID, 10); err != nil {
		t.Fatalf("adjust physical quantity: %v", err)
	}

	if err := repo.Reserve(ctx, productID, 8); err != nil {
		t.Fatalf("reserve quantity: %v", err)
	}

	if err := repo.Release(ctx, productID, 3); err != nil {
		t.Fatalf("release quantity: %v", err)
	}

	got, err := repo.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != 10 {
		t.Errorf("physical quantity = %d, want 10", got.PhysicalQuantity)
	}

	if got.ReservedQuantity != 5 {
		t.Errorf("reserved quantity = %d, want 5", got.ReservedQuantity)
	}

	if got.AvailableQuantity() != 5 {
		t.Errorf("available quantity = %d, want 5", got.AvailableQuantity())
	}
}

func TestPostgresRepository_Release_ExactReservedQuantity(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	productID := createTestProduct(t, ctx, tx)
	repo := NewPostgresRepository(tx)

	if err := repo.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	if err := repo.AdjustPhysicalQuantity(ctx, productID, 10); err != nil {
		t.Fatalf("adjust physical quantity: %v", err)
	}

	if err := repo.Reserve(ctx, productID, 8); err != nil {
		t.Fatalf("reserve quantity: %v", err)
	}

	if err := repo.Release(ctx, productID, 8); err != nil {
		t.Fatalf("release exact reserved quantity: %v", err)
	}

	got, err := repo.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.ReservedQuantity != 0 {
		t.Errorf("reserved quantity = %d, want 0", got.ReservedQuantity)
	}

	if got.AvailableQuantity() != 10 {
		t.Errorf("available quantity = %d, want 10", got.AvailableQuantity())
	}
}

func TestPostgresRepository_Release_InsufficientReservedStock(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	productID := createTestProduct(t, ctx, tx)
	repo := NewPostgresRepository(tx)

	if err := repo.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	if err := repo.AdjustPhysicalQuantity(ctx, productID, 10); err != nil {
		t.Fatalf("adjust physical quantity: %v", err)
	}

	if err := repo.Reserve(ctx, productID, 5); err != nil {
		t.Fatalf("reserve quantity: %v", err)
	}

	err = repo.Release(ctx, productID, 6)
	if !errors.Is(err, ErrInsufficientReservedStock) {
		t.Fatalf("release error = %v, want %v", err, ErrInsufficientReservedStock)
	}

	got, err := repo.GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.ReservedQuantity != 5 {
		t.Errorf("reserved quantity = %d, want 5", got.ReservedQuantity)
	}
}

func TestPostgresRepository_Release_ZeroQuantity(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	productID := createTestProduct(t, ctx, tx)
	repo := NewPostgresRepository(tx)

	if err := repo.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	err = repo.Release(ctx, productID, 0)
	if !errors.Is(err, ErrInvalidReservation) {
		t.Fatalf("release error = %v, want %v", err, ErrInvalidReservation)
	}
}

func TestPostgresRepository_Release_NegativeQuantity(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	productID := createTestProduct(t, ctx, tx)
	repo := NewPostgresRepository(tx)

	if err := repo.Create(ctx, productID); err != nil {
		t.Fatalf("create inventory: %v", err)
	}

	err = repo.Release(ctx, productID, -1)
	if !errors.Is(err, ErrInvalidReservation) {
		t.Fatalf("release error = %v, want %v", err, ErrInvalidReservation)
	}
}

func cleanupCommittedInventory(t *testing.T, db *sql.DB, productID int64) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := db.ExecContext(ctx, `DELETE FROM inventory WHERE product_id = $1`, productID); err != nil {
			t.Errorf("delete test inventory: %v", err)
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM products WHERE id = $1`, productID); err != nil {
			t.Errorf("delete test product: %v", err)
		}
	})
}

func TestPostgresRepository_Reserve_ConcurrentRequests(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// Create inventory using a transaction.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}

	productID := createTestProduct(t, ctx, tx)
	setupRepo := NewPostgresRepository(tx)

	if err := setupRepo.Create(ctx, productID); err != nil {
		tx.Rollback()
		t.Fatalf("create inventory: %v", err)
	}

	if err := setupRepo.AdjustPhysicalQuantity(ctx, productID, 10); err != nil {
		tx.Rollback()
		t.Fatalf("adjust physical quantity: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit setup transaction: %v", err)
	}
	cleanupCommittedInventory(t, db, productID)

	// Both goroutines will start their reservation at approximately
	// the same time.
	start := make(chan struct{})

	results := make(chan error, 2)

	reserve := func() {
		<-start

		repo := NewPostgresRepository(db)
		results <- repo.Reserve(ctx, productID, 7)
	}

	go reserve()
	go reserve()

	// Release both goroutines.
	close(start)

	err1 := <-results
	err2 := <-results

	// Exactly one request must succeed.
	successCount := 0
	failureCount := 0

	for _, err := range []error{err1, err2} {
		if err == nil {
			successCount++
			continue
		}

		if errors.Is(err, ErrInsufficientStock) {
			failureCount++
			continue
		}

		t.Fatalf("unexpected reservation error: %v", err)
	}

	if successCount != 1 {
		t.Errorf("successful reservations = %d, want 1", successCount)
	}

	if failureCount != 1 {
		t.Errorf("failed reservations = %d, want 1", failureCount)
	}

	// Verify the persisted inventory state.
	got, err := NewPostgresRepository(db).GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != 10 {
		t.Errorf("physical quantity = %d, want 10", got.PhysicalQuantity)
	}

	if got.ReservedQuantity != 7 {
		t.Errorf("reserved quantity = %d, want 7", got.ReservedQuantity)
	}

	if got.AvailableQuantity() != 3 {
		t.Errorf("available quantity = %d, want 3", got.AvailableQuantity())
	}
}

func TestPostgresRepository_Reserve_HighConcurrency(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	const (
		physicalQuantity int32 = 100
		reservationQty   int32 = 10
		requestCount           = 20
	)

	// Set up inventory.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}

	productID := createTestProduct(t, ctx, tx)
	setupRepo := NewPostgresRepository(tx)

	if err := setupRepo.Create(ctx, productID); err != nil {
		tx.Rollback()
		t.Fatalf("create inventory: %v", err)
	}

	if err := setupRepo.AdjustPhysicalQuantity(ctx, productID, physicalQuantity); err != nil {
		tx.Rollback()
		t.Fatalf("adjust physical quantity: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit setup transaction: %v", err)
	}
	cleanupCommittedInventory(t, db, productID)

	// Prepare concurrent reservation requests.
	start := make(chan struct{})
	results := make(chan error, requestCount)

	for i := 0; i < requestCount; i++ {
		go func() {
			<-start

			repo := NewPostgresRepository(db)
			results <- repo.Reserve(ctx, productID, reservationQty)
		}()
	}

	// Release all requests together.
	close(start)

	// Collect results.
	var successCount int
	var failureCount int

	for i := 0; i < requestCount; i++ {
		err := <-results

		if err == nil {
			successCount++
			continue
		}

		if errors.Is(err, ErrInsufficientStock) {
			failureCount++
			continue
		}

		t.Errorf("unexpected reservation error: %v", err)
	}

	// Exactly ten requests should succeed.
	if successCount != 10 {
		t.Errorf("successful reservations = %d, want 10", successCount)
	}

	if failureCount != 10 {
		t.Errorf("failed reservations = %d, want 10", failureCount)
	}

	// Verify final persisted state.
	got, err := NewPostgresRepository(db).GetByProductID(ctx, productID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}

	if got.PhysicalQuantity != 100 {
		t.Errorf("physical quantity = %d, want 100", got.PhysicalQuantity)
	}

	if got.ReservedQuantity != 100 {
		t.Errorf("reserved quantity = %d, want 100", got.ReservedQuantity)
	}

	if got.AvailableQuantity() != 0 {
		t.Errorf("available quantity = %d, want 0", got.AvailableQuantity())
	}
}
