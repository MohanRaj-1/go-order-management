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
func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	healthHandler(rec, req)

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
	newRouter(customerHandler).ServeHTTP(rec, req)

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
	mux := newRouter(customerHandler)

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
	mux := newRouter(customerHandler)

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
