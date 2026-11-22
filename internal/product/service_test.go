package product

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/MohanRaj-1/go-order-management/internal/database"
	"github.com/jackc/pgx/v5/pgconn"
)

func setupServiceProductTest(t *testing.T) (context.Context, *sql.DB, *Service, string) {
	t.Helper()

	connectionString := os.Getenv("TEST_DATABASE_URL")
	if connectionString == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	db, err := database.Open(ctx, connectionString, database.DefaultConfig())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	service := NewService(db, database.NewTransactionManager(db))
	sku := fmt.Sprintf("product-service-test-%d", time.Now().UnixNano())

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := db.ExecContext(ctx, `
			DELETE FROM inventory
			WHERE product_id IN (SELECT id FROM products WHERE sku = $1)
		`, sku)
		if err != nil {
			t.Errorf("clean up test inventory: %v", err)
		}

		if _, err := db.ExecContext(ctx, "DELETE FROM products WHERE sku = $1", sku); err != nil {
			t.Errorf("clean up test product: %v", err)
		}
	})

	return ctx, db, service, sku
}

func TestService_CreateProduct_Success(t *testing.T) {
	ctx, db, service, sku := setupServiceProductTest(t)
	product, err := service.CreateProduct(ctx, sku, "Service Test Product", "Test description", 1250)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if product == nil {
		t.Fatal("expected a product")
	}
	if product.ID <= 0 {
		t.Fatalf("expected a positive product ID, got %d", product.ID)
	}
	if product.CreatedAt.IsZero() {
		t.Fatal("expected product CreatedAt to be populated")
	}
	if product.UpdatedAt.IsZero() {
		t.Fatal("expected product UpdatedAt to be populated")
	}

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM products WHERE id = $1", product.ID).Scan(&count)
	if err != nil {
		t.Fatalf("query committed product: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 committed product, got %d", count)
	}

	var physicalQuantity, reservedQuantity int
	err = db.QueryRowContext(ctx, `
		SELECT physical_quantity, reserved_quantity
		FROM inventory
		WHERE product_id = $1
	`, product.ID).Scan(&physicalQuantity, &reservedQuantity)
	if err != nil {
		t.Fatalf("query initial inventory: %v", err)
	}
	if physicalQuantity != 0 || reservedQuantity != 0 {
		t.Fatalf("expected zero inventory quantities, got physical=%d and reserved=%d",
			physicalQuantity, reservedQuantity)
	}
}

func TestService_GetProductByID_Success(t *testing.T) {
	ctx, _, service, sku := setupServiceProductTest(t)
	created, err := service.CreateProduct(ctx, sku, "Service Lookup Product", "Lookup test description", 1250)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	if created == nil {
		t.Fatal("expected a created product")
	}

	got, err := service.GetProductByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get product by ID %d: %v", created.ID, err)
	}
	if got == nil {
		t.Fatal("expected a retrieved product")
	}
	if got.ID != created.ID || got.SKU != created.SKU || got.Name != created.Name ||
		got.Description != created.Description || got.Price != created.Price || got.Status != created.Status ||
		!got.CreatedAt.Equal(created.CreatedAt) || !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("retrieved product = %+v, want %+v", *got, *created)
	}
}

func TestService_GetProductByID_NotFound(t *testing.T) {
	_, _, service, _ := setupServiceProductTest(t)

	product, err := service.GetProductByID(context.Background(), -1)

	if err == nil {
		t.Fatal("expected an error for a non-existent product")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if product != nil {
		t.Fatalf("expected nil product, got %+v", product)
	}
}

func TestService_GetProductByID_DatabaseError(t *testing.T) {
	_, _, service, _ := setupServiceProductTest(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	product, err := service.GetProductByID(ctx, 1)

	if err == nil {
		t.Fatal("expected a database error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if errors.Is(err, ErrNotFound) {
		t.Fatalf("expected a database error, got ErrNotFound: %v", err)
	}

	if product != nil {
		t.Fatalf("expected nil product, got %+v", product)
	}
}

func TestService_CreateProduct_RollsBackWhenInventoryCreationFails(t *testing.T) {
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

	sku := fmt.Sprintf("product-rollback-test-%d", time.Now().UnixNano())
	manager := database.NewTransactionManager(db)

	// Install both objects atomically before the service starts its transaction.
	err = manager.WithinTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			CREATE FUNCTION test_fail_inventory_insert()
			RETURNS TRIGGER AS $$
			BEGIN
				IF EXISTS (SELECT 1 FROM products WHERE id = NEW.product_id AND sku = TG_ARGV[0]) THEN
					RAISE EXCEPTION 'simulated inventory insertion failure';
				END IF;
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;
		`)
		if err != nil {
			return fmt.Errorf("create test trigger function: %w", err)
		}

		// The generated SKU contains only a fixed prefix and a decimal integer.
		_, err = tx.ExecContext(ctx, fmt.Sprintf(`
			CREATE TRIGGER test_inventory_insert_failure
			BEFORE INSERT ON inventory
			FOR EACH ROW
			EXECUTE FUNCTION test_fail_inventory_insert('%s');
		`, sku))
		return err
	})
	if err != nil {
		t.Fatalf("install inventory failure trigger: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()

		if _, err := db.ExecContext(cleanupCtx,
			`DROP TRIGGER IF EXISTS test_inventory_insert_failure ON inventory`,
		); err != nil {
			t.Errorf("drop test trigger: %v", err)
		}
		if _, err := db.ExecContext(cleanupCtx,
			`DROP FUNCTION IF EXISTS test_fail_inventory_insert()`,
		); err != nil {
			t.Errorf("drop test trigger function: %v", err)
		}

		if _, err := db.ExecContext(cleanupCtx, `
			DELETE FROM inventory
			WHERE product_id IN (SELECT id FROM products WHERE sku = $1)
		`, sku); err != nil {
			t.Errorf("clean up test inventory: %v", err)
		}
		if _, err := db.ExecContext(cleanupCtx, "DELETE FROM products WHERE sku = $1", sku); err != nil {
			t.Errorf("clean up test product: %v", err)
		}
	})

	service := NewService(db, manager)
	product, err := service.CreateProduct(ctx, sku, "Rollback Test Product", "Test description", 1250)
	if err == nil {
		t.Fatal("expected inventory creation to fail")
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "P0001" || pgErr.Message != "simulated inventory insertion failure" {
		t.Fatalf("expected inventory trigger exception, got %v", err)
	}
	if product != nil {
		t.Fatal("expected nil product after transaction rollback")
	}

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM products WHERE sku = $1", sku).Scan(&count)
	if err != nil {
		t.Fatalf("query product after rollback: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected product insertion to be rolled back, got %d products", count)
	}
}

func TestService_CreateProduct_InvalidInput(t *testing.T) {
	service := NewService(nil, nil)

	product, err := service.CreateProduct(
		context.Background(),
		"",
		"Valid Product",
		"Description",
		1250,
	)

	if !errors.Is(err, ErrInvalidSKU) {
		t.Fatalf("expected ErrInvalidSKU, got %v", err)
	}

	if product != nil {
		t.Fatal("expected nil product for invalid input")
	}
}
