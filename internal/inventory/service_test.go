package inventory

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type mockRepository struct {
	createCalls                     int
	getByProductIDCalls             int
	getByProductIDResult            *Inventory
	getByProductIDErr               error
	adjustPhysicalQuantityCalls     int
	adjustPhysicalQuantityProductID int64
	adjustPhysicalQuantityDelta     int32
	adjustPhysicalQuantityErr       error
	reserveCalls                    int
	reserveProductID                int64
	reserveQuantity                 int32
	reserveErr                      error
	releaseCalls                    int
	releaseProductID                int64
	releaseQuantity                 int32
	releaseErr                      error
}

var _ Repository = (*mockRepository)(nil)

func (m *mockRepository) Create(ctx context.Context, productID int64) error {
	m.createCalls++
	return nil
}

func (m *mockRepository) GetByProductID(ctx context.Context, productID int64) (*Inventory, error) {
	m.getByProductIDCalls++
	return m.getByProductIDResult, m.getByProductIDErr
}

func (m *mockRepository) AdjustPhysicalQuantity(ctx context.Context, productID int64, delta int32) error {
	m.adjustPhysicalQuantityCalls++
	m.adjustPhysicalQuantityProductID = productID
	m.adjustPhysicalQuantityDelta = delta
	return m.adjustPhysicalQuantityErr
}

func (m *mockRepository) Reserve(ctx context.Context, productID int64, quantity int32) error {
	m.reserveCalls++
	m.reserveProductID = productID
	m.reserveQuantity = quantity
	return m.reserveErr
}

func (m *mockRepository) Release(ctx context.Context, productID int64, quantity int32) error {
	m.releaseCalls++
	m.releaseProductID = productID
	m.releaseQuantity = quantity
	return m.releaseErr
}

func TestService_AdjustPhysicalQuantity(t *testing.T) {
	const (
		productID int64 = 10
		delta     int32 = 15
	)

	repository := &mockRepository{}
	service := NewService(repository)

	if err := service.AdjustPhysicalQuantity(context.Background(), productID, delta); err != nil {
		t.Fatalf("AdjustPhysicalQuantity returned an error: %v", err)
	}
	if repository.adjustPhysicalQuantityCalls != 1 {
		t.Errorf("AdjustPhysicalQuantity calls = %d, want 1", repository.adjustPhysicalQuantityCalls)
	}
	if repository.adjustPhysicalQuantityProductID != productID {
		t.Errorf("AdjustPhysicalQuantity product ID = %d, want %d", repository.adjustPhysicalQuantityProductID, productID)
	}
	if repository.adjustPhysicalQuantityDelta != delta {
		t.Errorf("AdjustPhysicalQuantity delta = %d, want %d", repository.adjustPhysicalQuantityDelta, delta)
	}
}

func TestService_AdjustPhysicalQuantity_InvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		productID int64
		delta     int32
		wantErr   error
	}{
		{name: "zero product ID", productID: 0, delta: 10, wantErr: ErrInvalidProductID},
		{name: "negative product ID", productID: -1, delta: 10, wantErr: ErrInvalidProductID},
		{name: "zero delta", productID: 10, delta: 0, wantErr: ErrInvalidAdjustment},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &mockRepository{}
			service := NewService(repository)

			err := service.AdjustPhysicalQuantity(context.Background(), tt.productID, tt.delta)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("AdjustPhysicalQuantity error = %v, want %v", err, tt.wantErr)
			}
			if repository.adjustPhysicalQuantityCalls != 0 {
				t.Errorf("AdjustPhysicalQuantity calls = %d, want 0", repository.adjustPhysicalQuantityCalls)
			}
		})
	}
}

func TestService_AdjustPhysicalQuantity_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("repository adjustment failed")
	repository := &mockRepository{adjustPhysicalQuantityErr: repositoryErr}
	service := NewService(repository)

	err := service.AdjustPhysicalQuantity(context.Background(), 10, 15)
	if err == nil {
		t.Fatal("AdjustPhysicalQuantity returned nil error")
	}
	if !errors.Is(err, repositoryErr) {
		t.Errorf("AdjustPhysicalQuantity error = %v, want an error wrapping %v", err, repositoryErr)
	}
	if err == repositoryErr {
		t.Error("AdjustPhysicalQuantity returned the repository error without wrapping it")
	}
	if repository.adjustPhysicalQuantityCalls != 1 {
		t.Errorf("AdjustPhysicalQuantity calls = %d, want 1", repository.adjustPhysicalQuantityCalls)
	}
}

func TestService_AdjustPhysicalQuantity_InsufficientPhysicalStock(t *testing.T) {
	repository := &mockRepository{adjustPhysicalQuantityErr: ErrInsufficientPhysicalStock}
	service := NewService(repository)

	err := service.AdjustPhysicalQuantity(context.Background(), 10, -1)
	if !errors.Is(err, ErrInsufficientPhysicalStock) {
		t.Errorf("AdjustPhysicalQuantity error = %v, want %v", err, ErrInsufficientPhysicalStock)
	}
	if repository.adjustPhysicalQuantityCalls != 1 {
		t.Errorf("AdjustPhysicalQuantity calls = %d, want 1", repository.adjustPhysicalQuantityCalls)
	}
}

func TestService_ReserveStock(t *testing.T) {
	const (
		productID int64 = 10
		quantity  int32 = 5
	)

	repository := &mockRepository{}
	service := NewService(repository)

	if err := service.ReserveStock(context.Background(), productID, quantity); err != nil {
		t.Fatalf("ReserveStock returned an error: %v", err)
	}
	if repository.reserveCalls != 1 {
		t.Errorf("Reserve calls = %d, want 1", repository.reserveCalls)
	}
	if repository.reserveProductID != productID {
		t.Errorf("Reserve product ID = %d, want %d", repository.reserveProductID, productID)
	}
	if repository.reserveQuantity != quantity {
		t.Errorf("Reserve quantity = %d, want %d", repository.reserveQuantity, quantity)
	}
}

func TestService_ReserveStock_InvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		productID int64
		quantity  int32
		wantErr   error
	}{
		{name: "zero product ID", productID: 0, quantity: 5, wantErr: ErrInvalidProductID},
		{name: "negative product ID", productID: -1, quantity: 5, wantErr: ErrInvalidProductID},
		{name: "zero quantity", productID: 10, quantity: 0, wantErr: ErrInvalidReservation},
		{name: "negative quantity", productID: 10, quantity: -5, wantErr: ErrInvalidReservation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &mockRepository{}
			service := NewService(repository)

			err := service.ReserveStock(context.Background(), tt.productID, tt.quantity)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ReserveStock error = %v, want %v", err, tt.wantErr)
			}
			if repository.reserveCalls != 0 {
				t.Errorf("Reserve calls = %d, want 0", repository.reserveCalls)
			}
		})
	}
}

func TestService_ReserveStock_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("repository reservation failed")
	repository := &mockRepository{reserveErr: repositoryErr}
	service := NewService(repository)

	err := service.ReserveStock(context.Background(), 10, 5)
	if err == nil {
		t.Fatal("ReserveStock returned nil error")
	}
	if !errors.Is(err, repositoryErr) {
		t.Errorf("ReserveStock error = %v, want an error wrapping %v", err, repositoryErr)
	}
	if err == repositoryErr {
		t.Error("ReserveStock returned the repository error without wrapping it")
	}
	if repository.reserveCalls != 1 {
		t.Errorf("Reserve calls = %d, want 1", repository.reserveCalls)
	}
}

func TestService_ReleaseStock(t *testing.T) {
	const (
		productID int64 = 10
		quantity  int32 = 5
	)

	repository := &mockRepository{}
	service := NewService(repository)

	if err := service.ReleaseStock(context.Background(), productID, quantity); err != nil {
		t.Fatalf("ReleaseStock returned an error: %v", err)
	}
	if repository.releaseCalls != 1 {
		t.Errorf("Release calls = %d, want 1", repository.releaseCalls)
	}
	if repository.releaseProductID != productID {
		t.Errorf("Release product ID = %d, want %d", repository.releaseProductID, productID)
	}
	if repository.releaseQuantity != quantity {
		t.Errorf("Release quantity = %d, want %d", repository.releaseQuantity, quantity)
	}
}

func TestService_ReleaseStock_InvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		productID int64
		quantity  int32
		wantErr   error
	}{
		{name: "zero product ID", productID: 0, quantity: 5, wantErr: ErrInvalidProductID},
		{name: "negative product ID", productID: -1, quantity: 5, wantErr: ErrInvalidProductID},
		{name: "zero quantity", productID: 10, quantity: 0, wantErr: ErrInvalidReservation},
		{name: "negative quantity", productID: 10, quantity: -5, wantErr: ErrInvalidReservation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &mockRepository{}
			service := NewService(repository)

			err := service.ReleaseStock(context.Background(), tt.productID, tt.quantity)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ReleaseStock error = %v, want %v", err, tt.wantErr)
			}
			if repository.releaseCalls != 0 {
				t.Errorf("Release calls = %d, want 0", repository.releaseCalls)
			}
		})
	}
}

func TestService_ReleaseStock_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("repository release failed")
	repository := &mockRepository{releaseErr: repositoryErr}
	service := NewService(repository)

	err := service.ReleaseStock(context.Background(), 10, 5)
	if err == nil {
		t.Fatal("ReleaseStock returned nil error")
	}
	if !errors.Is(err, repositoryErr) {
		t.Errorf("ReleaseStock error = %v, want an error wrapping %v", err, repositoryErr)
	}
	if err == repositoryErr {
		t.Error("ReleaseStock returned the repository error without wrapping it")
	}
	if repository.releaseCalls != 1 {
		t.Errorf("Release calls = %d, want 1", repository.releaseCalls)
	}
}

func TestService_GetInventory(t *testing.T) {
	want := &Inventory{
		ID:               1,
		ProductID:        10,
		PhysicalQuantity: 20,
		ReservedQuantity: 5,
	}
	repository := &mockRepository{getByProductIDResult: want}
	service := NewService(repository)

	got, err := service.GetInventory(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetInventory returned an error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetInventory result = %+v, want %+v", got, want)
	}

	if repository.getByProductIDCalls != 1 {
		t.Errorf("GetByProductID calls = %d, want 1", repository.getByProductIDCalls)
	}
}

func TestService_GetInventory_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("repository lookup failed")
	repository := &mockRepository{getByProductIDErr: repositoryErr}
	service := NewService(repository)

	_, err := service.GetInventory(context.Background(), 10)
	if err == nil {
		t.Error("GetInventory returned nil error")
	} else if !errors.Is(err, repositoryErr) {
		t.Errorf("GetInventory error = %v, want an error wrapping %v", err, repositoryErr)
	} else if err == repositoryErr {
		t.Error("GetInventory returned the repository error without wrapping it")
	}
	if repository.getByProductIDCalls != 1 {
		t.Errorf("GetByProductID calls = %d, want 1", repository.getByProductIDCalls)
	}
}

func TestService_GetInventory_InvalidProductID(t *testing.T) {
	tests := []struct {
		name      string
		productID int64
	}{
		{name: "zero", productID: 0},
		{name: "negative", productID: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &mockRepository{}
			service := NewService(repository)

			_, err := service.GetInventory(context.Background(), tt.productID)
			if !errors.Is(err, ErrInvalidProductID) {
				t.Errorf("GetInventory error = %v, want %v", err, ErrInvalidProductID)
			}
			if repository.getByProductIDCalls != 0 {
				t.Errorf("GetByProductID calls = %d, want 0", repository.getByProductIDCalls)
			}
		})
	}
}
