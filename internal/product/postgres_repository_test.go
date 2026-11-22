package product

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/MohanRaj-1/go-order-management/internal/database"
)

func TestPostgresRepository_GetByID_Success(t *testing.T) {
	connectionString := os.Getenv("TEST_DATABASE_URL")
	if connectionString == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Open(ctx, connectionString, database.DefaultConfig())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	repository := NewPostgresRepository(db)
	sku := fmt.Sprintf("product-repository-test-%d", time.Now().UnixNano())

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cleanupCancel()

		if _, err := db.ExecContext(
			cleanupCtx,
			"DELETE FROM products WHERE sku = $1",
			sku,
		); err != nil {
			t.Errorf("clean up test product: %v", err)
		}
	})

	original, err := NewProduct(
		sku,
		"Repository Test Product",
		"Testing product retrieval",
		1250,
	)
	if err != nil {
		t.Fatalf("create product model: %v", err)
	}

	if err := repository.Create(ctx, original); err != nil {
		t.Fatalf("insert test product: %v", err)
	}

	retrieved, err := repository.GetByID(ctx, original.ID)
	if err != nil {
		t.Fatalf("get product by ID: %v", err)
	}

	if retrieved == nil {
		t.Fatal("expected a product")
	}

	if retrieved.ID != original.ID {
		t.Errorf("expected ID %d, got %d", original.ID, retrieved.ID)
	}

	if retrieved.SKU != original.SKU {
		t.Errorf("expected SKU %q, got %q", original.SKU, retrieved.SKU)
	}

	if retrieved.Name != original.Name {
		t.Errorf("expected name %q, got %q", original.Name, retrieved.Name)
	}

	if retrieved.Description != original.Description {
		t.Errorf("expected description %q, got %q",
			original.Description, retrieved.Description)
	}

	if retrieved.Price != original.Price {
		t.Errorf("expected price %d paise, got %d paise",
			original.Price, retrieved.Price)
	}

	if retrieved.Status != original.Status {
		t.Errorf("expected status %q, got %q",
			original.Status, retrieved.Status)
	}

	if !retrieved.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("expected CreatedAt %v, got %v",
			original.CreatedAt, retrieved.CreatedAt)
	}

	if !retrieved.UpdatedAt.Equal(original.UpdatedAt) {
		t.Errorf("expected UpdatedAt %v, got %v",
			original.UpdatedAt, retrieved.UpdatedAt)
	}
}

func TestPostgresRepository_GetByID_NotFound(t *testing.T) {
	connectionString := os.Getenv("TEST_DATABASE_URL")
	if connectionString == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Open(ctx, connectionString, database.DefaultConfig())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	}()

	repository := NewPostgresRepository(db)

	product, err := repository.GetByID(ctx, -1)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if product != nil {
		t.Fatalf("expected nil product, got %+v", product)
	}
}

func TestPostgresRepository_GetByID_DatabaseError(t *testing.T) {
	connectionString := os.Getenv("TEST_DATABASE_URL")
	if connectionString == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	dbCtx, dbCancel := context.WithTimeout(
		context.Background(), 5*time.Second,
	)
	defer dbCancel()

	db, err := database.Open(
		dbCtx, connectionString, database.DefaultConfig(),
	)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	}()

	repository := NewPostgresRepository(db)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	product, err := repository.GetByID(ctx, 1)

	if err == nil {
		t.Fatal("expected database error")
	}

	if errors.Is(err, ErrNotFound) {
		t.Fatalf("expected database error, got ErrNotFound: %v", err)
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if product != nil {
		t.Fatalf("expected nil product, got %+v", product)
	}
}
