package customer

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MohanRaj-1/go-order-management/internal/database"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
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

func TestPostgresRepository_Create(t *testing.T) {
	db := newTestDB(t)

	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repo := NewPostgresRepository(tx)

	customer := &Customer{
		Name:  "Integration Test Customer",
		Email: "integration-test@example.com",
	}

	err = repo.Create(ctx, customer)
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	if customer.ID <= 0 {
		t.Errorf("expected generated customer ID, got %d", customer.ID)
	}

	if customer.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be populated")
	}

	if customer.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be populated")
	}

	if customer.Name != "Integration Test Customer" {
		t.Errorf("unexpected customer name: %q", customer.Name)
	}

	if customer.Email != "integration-test@example.com" {
		t.Errorf("unexpected customer email: %q", customer.Email)
	}
}

func TestPostgresRepository_Create_DuplicateEmail(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repo := NewPostgresRepository(tx)
	first := &Customer{
		Name:  "First Customer",
		Email: "duplicate-email-test@example.com",
	}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create first customer: %v", err)
	}

	second := &Customer{
		Name:  "Second Customer",
		Email: first.Email,
	}
	if err := repo.Create(ctx, second); err == nil {
		t.Fatal("expected creating a customer with a duplicate email to fail")
	}
}

func TestPostgresRepository_Create_DuplicateEmail_CaseInsensitive(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repo := NewPostgresRepository(tx)
	first := &Customer{
		Name:  "First Customer",
		Email: "Case-Insensitive-Test@example.com",
	}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create first customer: %v", err)
	}

	second := &Customer{
		Name:  "Second Customer",
		Email: "case-insensitive-test@example.com",
	}
	if err := repo.Create(ctx, second); err == nil {
		t.Fatal("expected creating a customer with a differently cased duplicate email to fail")
	}
}

func TestPostgresRepository_GetByID(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repo := NewPostgresRepository(tx)
	want := &Customer{
		Name:  "GetByID Test Customer",
		Email: "get-by-id-test@example.com",
	}
	if err := repo.Create(ctx, want); err != nil {
		t.Fatalf("create customer: %v", err)
	}

	got, err := repo.GetByID(ctx, want.ID)
	if err != nil {
		t.Fatalf("get customer by ID: %v", err)
	}
	if got == nil {
		t.Fatal("expected customer, got nil")
	}

	if got.ID != want.ID {
		t.Errorf("ID = %d, want %d", got.ID, want.ID)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.Email != want.Email {
		t.Errorf("Email = %q, want %q", got.Email, want.Email)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, want.CreatedAt)
	}
	if !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, want.UpdatedAt)
	}
}

func TestPostgresRepository_GetByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer tx.Rollback()

	repo := NewPostgresRepository(tx)

	got, err := repo.GetByID(ctx, 999999999)

	if got != nil {
		t.Errorf("expected nil customer, got %+v", got)
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want %v", err, ErrNotFound)
	}
}
