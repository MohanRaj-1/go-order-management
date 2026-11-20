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

func TestService_CreateProduct_Success(t *testing.T) {
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

	service := NewService(database.NewTransactionManager(db))
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

	service := NewService(manager)
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
	service := NewService(nil)

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
