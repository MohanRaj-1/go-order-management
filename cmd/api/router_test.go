package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MohanRaj-1/go-order-management/internal/customer"
	httpapi "github.com/MohanRaj-1/go-order-management/internal/http"
	"github.com/MohanRaj-1/go-order-management/internal/product"
)

type stubCustomerService struct {
	createdCustomer *customer.Customer
	createErr       error
	createCalls     int
	gotName         string
	gotEmail        string

	getCustomer *customer.Customer
	getErr      error
	getCalls    int
	gotID       int64
}

type stubProductService struct {
	createdProduct *product.Product
	createErr      error
	createCalls    int

	foundProduct *product.Product
	getErr       error
	getCalls     int
	gotID        int64

	gotSKU         string
	gotName        string
	gotDescription string
	gotPrice       int64
}

func (s *stubProductService) CreateProduct(
	ctx context.Context,
	sku, name, description string,
	price int64,
) (*product.Product, error) {
	s.createCalls++
	s.gotSKU = sku
	s.gotName = name
	s.gotDescription = description
	s.gotPrice = price

	return s.createdProduct, s.createErr
}

func (s *stubProductService) GetProductByID(
	ctx context.Context,
	id int64,
) (*product.Product, error) {
	s.getCalls++
	s.gotID = id

	return s.foundProduct, s.getErr
}

func (s *stubCustomerService) CreateCustomer(
	ctx context.Context,
	name, email string,
) (*customer.Customer, error) {
	s.createCalls++
	s.gotName = name
	s.gotEmail = email

	return s.createdCustomer, s.createErr
}
func (s *stubCustomerService) GetCustomerByID(
	ctx context.Context,
	id int64,
) (*customer.Customer, error) {
	s.getCalls++
	s.gotID = id

	return s.getCustomer, s.getErr
}
func TestRouterHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	customerHandler := httpapi.NewCustomerHandler(&stubCustomerService{})
	productHandler := httpapi.NewProductHandler(&stubProductService{})
	mux := newRouter(customerHandler, productHandler)

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if got, want := rec.Body.String(), "OK\n"; got != want {
		t.Errorf("expected body %q, got %q", want, got)
	}
}

func TestRouterNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()

	customerHandler := httpapi.NewCustomerHandler(&stubCustomerService{})
	productHandler := httpapi.NewProductHandler(&stubProductService{})
	mux := newRouter(customerHandler, productHandler)

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestRouterCreateCustomer(t *testing.T) {
	now := time.Now().UTC()

	created := &customer.Customer{
		ID:        42,
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}

	service := &stubCustomerService{
		createdCustomer: created,
	}

	customerHandler := httpapi.NewCustomerHandler(service)
	productHandler := httpapi.NewProductHandler(&stubProductService{})
	mux := newRouter(customerHandler, productHandler)

	body := `{"name":"John Doe","email":"john@example.com"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/customers",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusCreated, rec.Code, rec.Body.String())
	}

	var got struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID != 42 || got.Name != "John Doe" || got.Email != "john@example.com" {
		t.Errorf("unexpected customer response: %+v", got)
	}

	if service.createCalls != 1 {
		t.Errorf("expected one service call, got %d", service.createCalls)
	}

	if service.gotName != "John Doe" || service.gotEmail != "john@example.com" {
		t.Errorf("unexpected service arguments: name=%q email=%q",
			service.gotName, service.gotEmail)
	}
}

func TestRouterGetCustomerByID(t *testing.T) {
	now := time.Now().UTC()

	expected := &customer.Customer{
		ID:        42,
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}

	service := &stubCustomerService{
		getCustomer: expected,
	}

	customerHandler := httpapi.NewCustomerHandler(service)
	productHandler := httpapi.NewProductHandler(&stubProductService{})
	mux := newRouter(customerHandler, productHandler)

	req := httptest.NewRequest(
		http.MethodGet,
		"/customers/42",
		nil,
	)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusOK, rec.Code, rec.Body.String())
	}

	var got struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID != expected.ID ||
		got.Name != expected.Name ||
		got.Email != expected.Email {
		t.Errorf("unexpected customer response: %+v", got)
	}

	if service.getCalls != 1 {
		t.Errorf("expected one service call, got %d", service.getCalls)
	}

	if service.gotID != 42 {
		t.Errorf("expected service ID 42, got %d", service.gotID)
	}
}

func TestRouterCreateProduct(t *testing.T) {
	now := time.Now().UTC()
	created := &product.Product{
		ID:          101,
		SKU:         "LAPTOP-001",
		Name:        "Laptop",
		Description: "Development laptop",
		Price:       7500000,
		Status:      product.ProductStatus("ACTIVE"),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	service := &stubProductService{createdProduct: created}
	customerHandler := httpapi.NewCustomerHandler(&stubCustomerService{})
	productHandler := httpapi.NewProductHandler(service)
	mux := newRouter(customerHandler, productHandler)

	body := `{"sku":"LAPTOP-001","name":"Laptop","description":"Development laptop","price":7500000}`
	req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusCreated, rec.Code, rec.Body.String())
	}

	var got struct {
		ID          int64                 `json:"id"`
		SKU         string                `json:"sku"`
		Name        string                `json:"name"`
		Description string                `json:"description"`
		Price       int64                 `json:"price"`
		Status      product.ProductStatus `json:"status"`
		CreatedAt   time.Time             `json:"created_at"`
		UpdatedAt   time.Time             `json:"updated_at"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != created.ID || got.SKU != created.SKU || got.Name != created.Name ||
		got.Description != created.Description || got.Price != created.Price || got.Status != created.Status ||
		!got.CreatedAt.Equal(created.CreatedAt) || !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("unexpected product response: %+v, want %+v", got, *created)
	}

	if service.createCalls != 1 {
		t.Errorf("expected one service call, got %d", service.createCalls)
	}
	if service.gotSKU != created.SKU || service.gotName != created.Name ||
		service.gotDescription != created.Description || service.gotPrice != created.Price {
		t.Errorf("unexpected service arguments: sku=%q name=%q description=%q price=%d",
			service.gotSKU, service.gotName, service.gotDescription, service.gotPrice)
	}
}

func TestRouterGetProductByID(t *testing.T) {
	now := time.Now().UTC()
	expected := &product.Product{
		ID:          101,
		SKU:         "LAPTOP-001",
		Name:        "Laptop",
		Description: "Development laptop",
		Price:       7500000,
		Status:      product.ProductStatus("ACTIVE"),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	service := &stubProductService{
		foundProduct: expected,
	}
	customerHandler := httpapi.NewCustomerHandler(&stubCustomerService{})
	productHandler := httpapi.NewProductHandler(service)
	mux := newRouter(customerHandler, productHandler)

	req := httptest.NewRequest(http.MethodGet, "/products/101", nil)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusOK, rec.Code, rec.Body.String())
	}

	var got struct {
		ID          int64                 `json:"id"`
		SKU         string                `json:"sku"`
		Name        string                `json:"name"`
		Description string                `json:"description"`
		Price       int64                 `json:"price"`
		Status      product.ProductStatus `json:"status"`
		CreatedAt   time.Time             `json:"created_at"`
		UpdatedAt   time.Time             `json:"updated_at"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != expected.ID || got.SKU != expected.SKU || got.Name != expected.Name ||
		got.Description != expected.Description || got.Price != expected.Price || got.Status != expected.Status ||
		!got.CreatedAt.Equal(expected.CreatedAt) || !got.UpdatedAt.Equal(expected.UpdatedAt) {
		t.Errorf("unexpected product response: %+v, want %+v", got, *expected)
	}

	if service.getCalls != 1 {
		t.Errorf("expected one service call, got %d", service.getCalls)
	}
	if service.gotID != 101 {
		t.Errorf("expected service ID 101, got %d", service.gotID)
	}
}
