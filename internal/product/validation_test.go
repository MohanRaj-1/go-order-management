package product

import (
	"errors"
	"strings"
	"testing"
)

func TestNewProduct(t *testing.T) {
	tests := []struct {
		name        string
		sku         string
		productName string
		price       int64
		wantErr     error
	}{
		{name: "valid product", sku: "SKU-123", productName: "Desk Lamp", price: 2499},
		{name: "SKU and name with surrounding spaces", sku: " SKU-123 ", productName: " Desk Lamp ", price: 2499},
		{name: "empty SKU", productName: "Desk Lamp", wantErr: ErrInvalidSKU},
		{name: "whitespace-only SKU", sku: " \t\n ", productName: "Desk Lamp", wantErr: ErrInvalidSKU},
		{name: "SKU exceeding 50 characters", sku: strings.Repeat("A", 51), productName: "Desk Lamp", wantErr: ErrInvalidSKU},
		{name: "SKU with 50 Unicode characters", sku: strings.Repeat("界", 50), productName: "Desk Lamp"},
		{name: "SKU with 51 Unicode characters", sku: strings.Repeat("界", 51), productName: "Desk Lamp", wantErr: ErrInvalidSKU},
		{name: "empty name", sku: "SKU-123", wantErr: ErrInvalidName},
		{name: "whitespace-only name", sku: "SKU-123", productName: " \t\n ", wantErr: ErrInvalidName},
		{name: "name with 150 characters", sku: "SKU-123", productName: strings.Repeat("a", 150)},
		{name: "name exceeding 150 characters", sku: "SKU-123", productName: strings.Repeat("a", 151), wantErr: ErrInvalidName},
		{name: "name with 150 Unicode characters", sku: "SKU-123", productName: strings.Repeat("界", 150)},
		{name: "name with 151 Unicode characters", sku: "SKU-123", productName: strings.Repeat("界", 151), wantErr: ErrInvalidName},
		{name: "zero price", sku: "SKU-123", productName: "Desk Lamp", price: 0},
		{name: "maximum price", sku: "SKU-123", productName: "Desk Lamp", price: 9_999_999_999},
		{name: "negative price", sku: "SKU-123", productName: "Desk Lamp", price: -1, wantErr: ErrInvalidPrice},
		{name: "price exceeding maximum", sku: "SKU-123", productName: "Desk Lamp", price: 10_000_000_000, wantErr: ErrInvalidPrice},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product, err := NewProduct(tt.sku, tt.productName, "", tt.price)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewProduct() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if product != nil {
					t.Errorf("NewProduct() = %+v, want nil on error", product)
				}
				return
			}
			if product == nil {
				t.Fatal("NewProduct() returned nil product")
			}
			if wantSKU := strings.TrimSpace(tt.sku); product.SKU != wantSKU {
				t.Errorf("product.SKU = %q, want %q", product.SKU, wantSKU)
			}
			if wantName := strings.TrimSpace(tt.productName); product.Name != wantName {
				t.Errorf("product.Name = %q, want %q", product.Name, wantName)
			}
			if product.Price != tt.price {
				t.Errorf("product.Price = %d, want %d", product.Price, tt.price)
			}
			if product.Status != ProductStatusActive {
				t.Errorf("product.Status = %q, want %q", product.Status, ProductStatusActive)
			}
		})
	}
}
