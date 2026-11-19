package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MohanRaj-1/go-order-management/internal/customer"
)

type fakeCustomerService struct {
	createdCustomer *customer.Customer
	createErr       error

	createCalls int
	gotName     string
	gotEmail    string

	getCustomer *customer.Customer
	getErr      error
	getCalls    int
	gotID       int64
}

var _ CustomerService = (*fakeCustomerService)(nil)

func (f *fakeCustomerService) CreateCustomer(
	ctx context.Context,
	name, email string,
) (*customer.Customer, error) {
	f.createCalls++
	f.gotName = name
	f.gotEmail = email

	return f.createdCustomer, f.createErr
}

func (f *fakeCustomerService) GetCustomerByID(
	ctx context.Context,
	id int64,
) (*customer.Customer, error) {
	f.getCalls++
	f.gotID = id

	return f.getCustomer, f.getErr
}

func TestCustomerHandler_GetCustomerByID_Success(t *testing.T) {
	want := customerResponse{
		ID:        42,
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Unix(100, 0).UTC(),
		UpdatedAt: time.Unix(200, 0).UTC(),
	}
	found := customer.Customer(want)
	service := &fakeCustomerService{getCustomer: &found}
	handler := NewCustomerHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/customers/42", nil)
	req.SetPathValue("id", "42")
	rec := httptest.NewRecorder()

	handler.GetCustomerByID(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusOK, rec.Code, rec.Body.String())
	}

	var got customerResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID != want.ID || got.Name != want.Name || got.Email != want.Email ||
		!got.CreatedAt.Equal(want.CreatedAt) || !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Errorf("response = %+v, want %+v", got, want)
	}

	if service.getCalls != 1 {
		t.Errorf("expected GetCustomerByID to be called once, got %d", service.getCalls)
	}
	if service.gotID != 42 {
		t.Errorf("expected service ID 42, got %d", service.gotID)
	}
}

func TestCustomerHandler_GetCustomerByID_NotFound(t *testing.T) {
	service := &fakeCustomerService{
		getErr: fmt.Errorf("get customer by ID: %w", customer.ErrNotFound),
	}
	handler := NewCustomerHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/customers/999", nil)
	req.SetPathValue("id", "999")
	rec := httptest.NewRecorder()

	handler.GetCustomerByID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusNotFound, rec.Code, rec.Body.String())
	}

	var got errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Error != "customer not found" {
		t.Errorf("expected error %q, got %q",
			"customer not found", got.Error)
	}

	if service.getCalls != 1 {
		t.Errorf("expected service to be called once, got %d",
			service.getCalls)
	}
}

func TestCustomerHandler_GetCustomerByID_InvalidID(t *testing.T) {
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
			service := &fakeCustomerService{}
			handler := NewCustomerHandler(service)

			req := httptest.NewRequest(
				http.MethodGet, "/customers/"+tt.id, nil,
			)
			req.SetPathValue("id", tt.id)

			rec := httptest.NewRecorder()
			handler.GetCustomerByID(rec, req)

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

func TestCustomerHandler_GetCustomerByID_InternalError(t *testing.T) {
	service := &fakeCustomerService{
		getErr: errors.New("database connection failed"),
	}
	handler := NewCustomerHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/customers/42", nil)
	req.SetPathValue("id", "42")
	rec := httptest.NewRecorder()

	handler.GetCustomerByID(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusInternalServerError, rec.Code, rec.Body.String())
	}

	var got errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Error != http.StatusText(http.StatusInternalServerError) {
		t.Errorf("expected generic error message, got %q", got.Error)
	}

	if service.getCalls != 1 {
		t.Errorf("expected service to be called once, got %d",
			service.getCalls)
	}
}

func TestCustomerHandler_CreateCustomer_Success(t *testing.T) {
	now := time.Now().UTC()

	expectedCustomer := &customer.Customer{
		ID:        1,
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}

	service := &fakeCustomerService{
		createdCustomer: expectedCustomer,
	}

	handler := NewCustomerHandler(service)

	body := `{"name":"John Doe","email":"john@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/customers", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusCreated, rec.Code, rec.Body.String())
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", contentType)
	}

	var got customerResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID != expectedCustomer.ID {
		t.Errorf("expected ID %d, got %d", expectedCustomer.ID, got.ID)
	}

	if got.Name != expectedCustomer.Name {
		t.Errorf("expected name %q, got %q", expectedCustomer.Name, got.Name)
	}

	if got.Email != expectedCustomer.Email {
		t.Errorf("expected email %q, got %q", expectedCustomer.Email, got.Email)
	}

	if !got.CreatedAt.Equal(expectedCustomer.CreatedAt) {
		t.Errorf("unexpected CreatedAt: got %v, want %v", got.CreatedAt, expectedCustomer.CreatedAt)
	}

	if !got.UpdatedAt.Equal(expectedCustomer.UpdatedAt) {
		t.Errorf("unexpected UpdatedAt: got %v, want %v", got.UpdatedAt, expectedCustomer.UpdatedAt)
	}

	if service.createCalls != 1 {
		t.Errorf("expected CreateCustomer to be called once, got %d", service.createCalls)
	}

	if service.gotName != "John Doe" || service.gotEmail != "john@example.com" {
		t.Errorf("unexpected service arguments: name=%q email=%q",
			service.gotName, service.gotEmail)
	}
}

func TestCustomerHandler_CreateCustomer_InvalidEmail(t *testing.T) {
	service := &fakeCustomerService{
		createErr: customer.ErrInvalidEmail,
	}

	handler := NewCustomerHandler(service)

	body := `{"name":"John Doe","email":"invalid-email"}`
	req := httptest.NewRequest(http.MethodPost, "/customers", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var got errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Error != customer.ErrInvalidEmail.Error() {
		t.Errorf("expected error %q, got %q",
			customer.ErrInvalidEmail.Error(), got.Error)
	}

	if service.createCalls != 1 {
		t.Errorf("expected CreateCustomer to be called once, got %d",
			service.createCalls)
	}
}

func TestCustomerHandler_CreateCustomer_InternalError(t *testing.T) {
	service := &fakeCustomerService{
		createErr: errors.New("database connection failed"),
	}

	handler := NewCustomerHandler(service)

	body := `{"name":"John Doe","email":"john@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/customers", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

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
}

func TestCustomerHandler_CreateCustomer_InvalidJSON(t *testing.T) {
	service := &fakeCustomerService{}
	handler := NewCustomerHandler(service)

	body := `{"name":"John Doe","email":}`
	req := httptest.NewRequest(http.MethodPost, "/customers", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d",
			http.StatusBadRequest, rec.Code)
	}

	var got errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Error != "invalid request body" {
		t.Errorf("expected invalid request body, got %q", got.Error)
	}

	if service.createCalls != 0 {
		t.Errorf("service should not be called, got %d calls", service.createCalls)
	}
}

func TestCustomerHandler_CreateCustomer_UnknownJSONField(t *testing.T) {
	service := &fakeCustomerService{}
	handler := NewCustomerHandler(service)

	body := `{"name":"John Doe","email":"john@example.com","phone":"1234567890"}`
	req := httptest.NewRequest(http.MethodPost, "/customers", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d",
			http.StatusBadRequest, rec.Code)
	}

	if service.createCalls != 0 {
		t.Errorf("service should not be called, got %d calls", service.createCalls)
	}
}

func TestCustomerHandler_CreateCustomer_MultipleJSONValues(t *testing.T) {
	service := &fakeCustomerService{}
	handler := NewCustomerHandler(service)

	body := `{"name":"John Doe","email":"john@example.com"} {"name":"Jane Doe","email":"jane@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/customers", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d",
			http.StatusBadRequest, rec.Code)
	}

	if service.createCalls != 0 {
		t.Errorf("service should not be called, got %d calls", service.createCalls)
	}

	var got errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Error != "request body must contain a single JSON object" {
		t.Errorf("unexpected error message: %q", got.Error)
	}
}

func TestCustomerHandler_CreateCustomer_RequestBodyTooLarge(t *testing.T) {
	service := &fakeCustomerService{}
	handler := NewCustomerHandler(service)

	body := `{"name":"` + strings.Repeat("a", maxRequestBodySize) + `","email":"john@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/customers", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d",
			http.StatusRequestEntityTooLarge, rec.Code)
	}

	if service.createCalls != 0 {
		t.Errorf("service should not be called, got %d calls", service.createCalls)
	}
}

func TestCustomerHandler_CreateCustomer_EmptyBody(t *testing.T) {
	service := &fakeCustomerService{}
	handler := NewCustomerHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/customers", nil)
	rec := httptest.NewRecorder()

	handler.CreateCustomer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d",
			http.StatusBadRequest, rec.Code)
	}

	if service.createCalls != 0 {
		t.Errorf("service should not be called, got %d calls", service.createCalls)
	}
}
