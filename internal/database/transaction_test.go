package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestTransactionManager_WithinTransaction_Commits(t *testing.T) {
	connectionString := os.Getenv("TEST_DATABASE_URL")
	if connectionString == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Open(ctx, connectionString, DefaultConfig())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	manager := NewTransactionManager(db)

	email := fmt.Sprintf(
		"transaction-test-%d@example.com",
		time.Now().UnixNano(),
	)

	t.Cleanup(func() {
		_, err := db.ExecContext(
			context.Background(),
			"DELETE FROM customers WHERE email = $1",
			email,
		)
		if err != nil {
			t.Errorf("clean up test customer: %v", err)
		}
	})

	err = manager.WithinTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO customers (name, email) VALUES ($1, $2)`,
			"Transaction Test",
			email,
		)
		return err
	})
	if err != nil {
		t.Fatalf("execute transaction: %v", err)
	}

	var count int
	err = db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM customers WHERE email = $1",
		email,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query committed customer: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 committed customer, got %d", count)
	}
}

func TestTransactionManager_WithinTransaction_RollsBackOnCallbackError(t *testing.T) {
	connectionString := os.Getenv("TEST_DATABASE_URL")
	if connectionString == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Open(ctx, connectionString, DefaultConfig())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	manager := NewTransactionManager(db)

	email := fmt.Sprintf(
		"transaction-rollback-test-%d@example.com",
		time.Now().UnixNano(),
	)

	t.Cleanup(func() {
		_, err := db.ExecContext(
			context.Background(),
			"DELETE FROM customers WHERE email = $1",
			email,
		)
		if err != nil {
			t.Errorf("clean up test customer: %v", err)
		}
	})

	expectedErr := errors.New("simulate operation failure")
	err = manager.WithinTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO customers (name, email) VALUES ($1, $2)`,
			"Transaction Test",
			email,
		)
		if err != nil {
			return err
		}
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected transaction error %v, got %v", expectedErr, err)
	}

	var count int
	err = db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM customers WHERE email = $1",
		email,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query customer after rollback: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected 0 customers after rollback, got %d", count)
	}
}

func TestTransactionManager_WithinTransaction_RollsBackOnPanic(t *testing.T) {
	connectionString := os.Getenv("TEST_DATABASE_URL")
	if connectionString == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Open(ctx, connectionString, DefaultConfig())
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	manager := NewTransactionManager(db)

	email := fmt.Sprintf(
		"transaction-panic-test-%d@example.com",
		time.Now().UnixNano(),
	)

	t.Cleanup(func() {
		_, err := db.ExecContext(
			context.Background(),
			"DELETE FROM customers WHERE email = $1",
			email,
		)
		if err != nil {
			t.Errorf("clean up test customer: %v", err)
		}
	})

	func() {
		defer func() {
			recovered := recover()
			if recovered != "simulate unexpected failure" {
				t.Fatalf(
					"expected panic %q, got %v",
					"simulate unexpected failure",
					recovered,
				)
			}
		}()

		_ = manager.WithinTransaction(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(
				ctx,
				`INSERT INTO customers (name, email) VALUES ($1, $2)`,
				"Transaction Test",
				email,
			)
			if err != nil {
				return err
			}

			panic("simulate unexpected failure")
		})
	}()

	var count int
	err = db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM customers WHERE email = $1",
		email,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query customer after panic: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected 0 customers after panic, got %d", count)
	}
}
