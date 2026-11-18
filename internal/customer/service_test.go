package customer

import (
	"context"
	"errors"
	"testing"
)

type fakeRepository struct {
	createFunc  func(ctx context.Context, customer *Customer) error
	getByIDFunc func(ctx context.Context, id int64) (*Customer, error)
}

func (f *fakeRepository) Create(
	ctx context.Context,
	customer *Customer,
) error {
	return f.createFunc(ctx, customer)
}

func (f *fakeRepository) GetByID(
	ctx context.Context,
	id int64,
) (*Customer, error) {
	return f.getByIDFunc(ctx, id)
}

func TestService_CreateCustomer_InvalidInput(t *testing.T) {
	repo := &fakeRepository{
		createFunc: func(ctx context.Context, customer *Customer) error {
			t.Fatal("repository Create should not be called for invalid input")
			return nil
		},
	}

	service := NewService(repo)

	got, err := service.CreateCustomer(
		context.Background(),
		"Alice",
		"invalid-email",
	)

	if !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("CreateCustomer() error = %v, want %v", err, ErrInvalidEmail)
	}

	if got != nil {
		t.Errorf("CreateCustomer() = %+v, want nil", got)
	}
}

func TestService_CreateCustomer_Success(t *testing.T) {
	repo := &fakeRepository{
		createFunc: func(ctx context.Context, customer *Customer) error {
			customer.ID = 1
			return nil
		},
	}

	service := NewService(repo)

	got, err := service.CreateCustomer(
		context.Background(),
		"Alice",
		"alice@example.com",
	)

	if err != nil {
		t.Fatalf("CreateCustomer() error = %v, want nil", err)
	}

	if got == nil {
		t.Fatal("CreateCustomer() returned nil customer")
	}

	if got.ID != 1 {
		t.Errorf("customer.ID = %d, want 1", got.ID)
	}

	if got.Name != "Alice" {
		t.Errorf("customer.Name = %q, want Alice", got.Name)
	}

	if got.Email != "alice@example.com" {
		t.Errorf("customer.Email = %q, want alice@example.com", got.Email)
	}
}

func TestService_CreateCustomer_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database connection failed")

	repo := &fakeRepository{
		createFunc: func(ctx context.Context, customer *Customer) error {
			return repositoryErr
		},
	}

	service := NewService(repo)

	got, err := service.CreateCustomer(
		context.Background(),
		"Alice",
		"alice@example.com",
	)

	if !errors.Is(err, repositoryErr) {
		t.Fatalf("CreateCustomer() error = %v, want %v", err, repositoryErr)
	}

	if got != nil {
		t.Errorf("CreateCustomer() = %+v, want nil", got)
	}
}

func TestService_GetCustomerByID_InvalidID(t *testing.T) {
	tests := []struct {
		name string
		id   int64
	}{
		{name: "zero", id: 0},
		{name: "negative", id: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{
				getByIDFunc: func(ctx context.Context, id int64) (*Customer, error) {
					t.Fatalf("repository GetByID(%d) should not be called for an invalid ID", id)
					return nil, nil
				},
			}

			service := NewService(repo)
			got, err := service.GetCustomerByID(context.Background(), tt.id)

			if !errors.Is(err, ErrInvalidCustomerID) {
				t.Errorf("GetCustomerByID(%d) error = %v, want %v", tt.id, err, ErrInvalidCustomerID)
			}

			if got != nil {
				t.Errorf("GetCustomerByID(%d) = %+v, want nil", tt.id, got)
			}
		})
	}
}

func TestService_GetCustomerByID_Success(t *testing.T) {
	const (
		wantID    int64 = 1
		wantName        = "Alice"
		wantEmail       = "alice@example.com"
	)

	repo := &fakeRepository{
		getByIDFunc: func(ctx context.Context, id int64) (*Customer, error) {
			if id != wantID {
				t.Fatalf("repository GetByID() called with ID %d, want %d", id, wantID)
			}

			return &Customer{ID: wantID, Name: wantName, Email: wantEmail}, nil
		},
	}

	service := NewService(repo)
	got, err := service.GetCustomerByID(context.Background(), wantID)
	if err != nil {
		t.Fatalf("GetCustomerByID() error = %v, want nil", err)
	}

	if got == nil {
		t.Fatal("GetCustomerByID() returned nil customer")
	}

	if got.ID != wantID {
		t.Errorf("customer.ID = %d, want %d", got.ID, wantID)
	}
	if got.Name != wantName {
		t.Errorf("customer.Name = %q, want %q", got.Name, wantName)
	}
	if got.Email != wantEmail {
		t.Errorf("customer.Email = %q, want %q", got.Email, wantEmail)
	}
}

func TestService_GetCustomerByID_NotFound(t *testing.T) {
	repo := &fakeRepository{
		getByIDFunc: func(ctx context.Context, id int64) (*Customer, error) {
			return nil, ErrNotFound
		},
	}

	service := NewService(repo)
	got, err := service.GetCustomerByID(context.Background(), 1)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetCustomerByID() error = %v, want %v", err, ErrNotFound)
	}

	if got != nil {
		t.Errorf("GetCustomerByID() = %+v, want nil", got)
	}
}
