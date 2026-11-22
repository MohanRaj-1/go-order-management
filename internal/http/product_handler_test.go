package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MohanRaj-1/go-order-management/internal/product"
)

type fakeProductService struct {
	createdProduct *product.Product
	createErr      error

	createCalls    int
	gotSKU         string
	gotName        string
	gotDescription string
	gotPrice       int64

	foundProduct *product.Product
	getErr       error

	getCalls int
	gotID    int64
}

var _ ProductService = (*fakeProductService)(nil)

func (f *fakeProductService) CreateProduct(
	ctx context.Context,
	sku string,
	name string,
	description string,
	price int64,
) (*product.Product, error) {
	f.createCalls++
	f.gotSKU = sku
	f.gotName = name
	f.gotDescription = description
	f.gotPrice = price

	return f.createdProduct, f.createErr
}

func (f *fakeProductService) GetProductByID(
	ctx context.Context,
	id int64,
) (*product.Product, error) {
	f.getCalls++
	f.gotID = id

	return f.foundProduct, f.getErr
}

func newTestProduct() *product.Product {
	now := time.Date(2026, time.January, 15, 10, 30, 0, 0, time.UTC)

	return &product.Product{
		ID:          42,
		SKU:         "LAPTOP-001",
		Name:        "Laptop",
		Description: "Development laptop",
		Price:       7500000,
		Status:      product.ProductStatus("ACTIVE"),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func TestProductHandler_GetProductByID_Success(t *testing.T) {
	want := newTestProduct()
	service := &fakeProductService{
		foundProduct: want,
	}
	handler := NewProductHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/products/42", nil)
	req.SetPathValue("id", "42")
	rec := httptest.NewRecorder()

	handler.GetProductByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusOK, rec.Code, rec.Body.String())
	}

	var got productResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID != want.ID ||
		got.SKU != want.SKU ||
		got.Name != want.Name ||
		got.Description != want.Description ||
		got.Price != want.Price ||
		got.Status != want.Status ||
		!got.CreatedAt.Equal(want.CreatedAt) ||
		!got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Errorf("response = %+v, want %+v", got, toProductResponse(want))
	}

	if service.getCalls != 1 {
		t.Errorf("expected GetProductByID to be called once, got %d",
			service.getCalls)
	}

	if service.gotID != 42 {
		t.Errorf("expected service ID 42, got %d", service.gotID)
	}
}

func TestProductHandler_GetProductByID_NotFound(t *testing.T) {
	service := &fakeProductService{
		getErr: product.ErrNotFound,
	}
	handler := NewProductHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/products/999", nil)
	req.SetPathValue("id", "999")
	rec := httptest.NewRecorder()

	handler.GetProductByID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusNotFound, rec.Code, rec.Body.String())
	}

	var got errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Error != "product not found" {
		t.Errorf("expected error %q, got %q",
			"product not found", got.Error)
	}

	if service.getCalls != 1 {
		t.Errorf("expected service to be called once, got %d",
			service.getCalls)
	}
}

func TestProductHandler_GetProductByID_InvalidID(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{name: "non-numeric ID", id: "abc"},
		{name: "zero ID", id: "0"},
		{name: "negative ID", id: "-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeProductService{}
			handler := NewProductHandler(service)

			req := httptest.NewRequest(
				http.MethodGet, "/products/"+tt.id, nil,
			)
			req.SetPathValue("id", tt.id)

			rec := httptest.NewRecorder()
			handler.GetProductByID(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d; body: %s",
					http.StatusBadRequest, rec.Code, rec.Body.String())
			}

			if service.getCalls != 0 {
				t.Errorf("service should not be called, got %d calls",
					service.getCalls)
			}
		})
	}
}

func TestProductHandler_GetProductByID_InternalError(t *testing.T) {
	service := &fakeProductService{
		getErr: errors.New("database connection failed"),
	}
	handler := NewProductHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/products/42", nil)
	req.SetPathValue("id", "42")
	rec := httptest.NewRecorder()

	handler.GetProductByID(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusInternalServerError, rec.Code, rec.Body.String())
	}

	var got errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Error != http.StatusText(http.StatusInternalServerError) {
		t.Errorf("expected generic error message %q, got %q",
			http.StatusText(http.StatusInternalServerError), got.Error)
	}

	if strings.Contains(got.Error, "database connection failed") {
		t.Error("internal error details must not be exposed")
	}

	if service.getCalls != 1 {
		t.Errorf("expected service to be called once, got %d",
			service.getCalls)
	}
}

func TestProductHandler_CreateProduct_Success(t *testing.T) {
	expectedProduct := newTestProduct()

	service := &fakeProductService{
		createdProduct: expectedProduct,
	}
	handler := NewProductHandler(service)

	body := `{
		"sku": "LAPTOP-001",
		"name": "Laptop",
		"description": "Development laptop",
		"price": 7500000
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/products",
		strings.NewReader(body),
	)
	rec := httptest.NewRecorder()

	handler.CreateProduct(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusCreated, rec.Code, rec.Body.String())
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", contentType)
	}

	var got productResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID != expectedProduct.ID ||
		got.SKU != expectedProduct.SKU ||
		got.Name != expectedProduct.Name ||
		got.Description != expectedProduct.Description ||
		got.Price != expectedProduct.Price ||
		got.Status != expectedProduct.Status ||
		!got.CreatedAt.Equal(expectedProduct.CreatedAt) ||
		!got.UpdatedAt.Equal(expectedProduct.UpdatedAt) {
		t.Errorf("response = %+v, want %+v",
			got, toProductResponse(expectedProduct))
	}

	if service.createCalls != 1 {
		t.Fatalf("expected CreateProduct to be called once, got %d",
			service.createCalls)
	}

	if service.gotSKU != "LAPTOP-001" ||
		service.gotName != "Laptop" ||
		service.gotDescription != "Development laptop" ||
		service.gotPrice != 7500000 {
		t.Errorf("unexpected service arguments: SKU=%q Name=%q Description=%q Price=%d",
			service.gotSKU,
			service.gotName,
			service.gotDescription,
			service.gotPrice,
		)
	}
}

func TestProductHandler_CreateProduct_ServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
		wantError  string
	}{
		{
			name:       "invalid SKU",
			serviceErr: product.ErrInvalidSKU,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid product SKU",
		},
		{
			name:       "invalid name",
			serviceErr: product.ErrInvalidName,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid product name",
		},
		{
			name:       "invalid price",
			serviceErr: product.ErrInvalidPrice,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid product price",
		},
		{
			name:       "internal error",
			serviceErr: errors.New("database connection failed"),
			wantStatus: http.StatusInternalServerError,
			wantError:  http.StatusText(http.StatusInternalServerError),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeProductService{
				createErr: tt.serviceErr,
			}
			handler := NewProductHandler(service)

			body := `{
				"sku": "LAPTOP-001",
				"name": "Laptop",
				"description": "Development laptop",
				"price": 7500000
			}`

			req := httptest.NewRequest(
				http.MethodPost,
				"/products",
				strings.NewReader(body),
			)
			rec := httptest.NewRecorder()

			handler.CreateProduct(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d; body: %s",
					tt.wantStatus, rec.Code, rec.Body.String())
			}

			var got errorResponse
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if got.Error != tt.wantError {
				t.Errorf("expected error %q, got %q",
					tt.wantError, got.Error)
			}

			if service.createCalls != 1 {
				t.Errorf("expected service to be called once, got %d",
					service.createCalls)
			}

			if tt.wantStatus == http.StatusInternalServerError &&
				strings.Contains(got.Error, "database connection failed") {
				t.Error("internal error details must not be exposed")
			}
		})
	}
}

func TestProductHandler_CreateProduct_InvalidJSON(t *testing.T) {
	service := &fakeProductService{}
	handler := NewProductHandler(service)

	body := `{"sku":"LAPTOP-001","name":}`

	req := httptest.NewRequest(
		http.MethodPost, "/products", strings.NewReader(body),
	)
	rec := httptest.NewRecorder()

	handler.CreateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var got errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Error != "invalid JSON request body" {
		t.Errorf("unexpected error message: %q", got.Error)
	}

	if service.createCalls != 0 {
		t.Errorf("service should not be called, got %d calls",
			service.createCalls)
	}
}

func TestProductHandler_CreateProduct_UnknownJSONField(t *testing.T) {
	service := &fakeProductService{}
	handler := NewProductHandler(service)

	body := `{
		"sku": "LAPTOP-001",
		"name": "Laptop",
		"description": "Development laptop",
		"price": 7500000,
		"manufacturer": "Example"
	}`

	req := httptest.NewRequest(
		http.MethodPost, "/products", strings.NewReader(body),
	)
	rec := httptest.NewRecorder()

	handler.CreateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	if service.createCalls != 0 {
		t.Errorf("service should not be called, got %d calls",
			service.createCalls)
	}
}

func TestProductHandler_CreateProduct_MultipleJSONValues(t *testing.T) {
	service := &fakeProductService{}
	handler := NewProductHandler(service)

	body := `{"sku":"LAPTOP-001","name":"Laptop","price":7500000} {"sku":"LAPTOP-002","name":"Desktop","price":5000000}`

	req := httptest.NewRequest(
		http.MethodPost, "/products", strings.NewReader(body),
	)
	rec := httptest.NewRecorder()

	handler.CreateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var got errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Error != "request body must contain a single JSON value" {
		t.Errorf("unexpected error message: %q", got.Error)
	}

	if service.createCalls != 0 {
		t.Errorf("service should not be called, got %d calls",
			service.createCalls)
	}
}

func TestProductHandler_CreateProduct_RequestBodyTooLarge(t *testing.T) {
	service := &fakeProductService{}
	handler := NewProductHandler(service)

	body := `{"sku":"` +
		strings.Repeat("a", maxRequestBodySize) +
		`","name":"Laptop","price":7500000}`

	req := httptest.NewRequest(
		http.MethodPost, "/products", strings.NewReader(body),
	)
	rec := httptest.NewRecorder()

	handler.CreateProduct(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
	}

	if service.createCalls != 0 {
		t.Errorf("service should not be called, got %d calls",
			service.createCalls)
	}
}

func TestProductHandler_CreateProduct_EmptyBody(t *testing.T) {
	service := &fakeProductService{}
	handler := NewProductHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/products", nil)
	rec := httptest.NewRecorder()

	handler.CreateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var got errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Error != "invalid JSON request body" {
		t.Errorf("unexpected error message: %q", got.Error)
	}

	if service.createCalls != 0 {
		t.Errorf("service should not be called, got %d calls",
			service.createCalls)
	}
}
