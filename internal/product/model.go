package product

import (
	"strings"
	"time"
)

type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "ACTIVE"
	ProductStatusInactive ProductStatus = "INACTIVE"
)

type Product struct {
	ID          int64
	SKU         string
	Name        string
	Description string
	Price       int64 // Price in minor units (paise)
	Status      ProductStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewProduct creates a product with its price represented in paise.
func NewProduct(
	sku string,
	name string,
	description string,
	price int64,
) (*Product, error) {
	sku = strings.TrimSpace(sku)
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if err := validateSKU(sku); err != nil {
		return nil, err
	}

	if err := validateName(name); err != nil {
		return nil, err
	}

	if err := validatePrice(price); err != nil {
		return nil, err
	}

	return &Product{
		SKU:         sku,
		Name:        name,
		Description: description,
		Price:       price,
		Status:      ProductStatusActive,
	}, nil
}
