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

	"github.com/MohanRaj-1/go-order-management/internal/inventory"
)

type getInventoryServiceStub struct {
	InventoryService
	found        *inventory.Inventory
	err          error
	calls        int
	gotProductID int64
}

func (s *getInventoryServiceStub) GetInventory(_ context.Context, productID int64) (*inventory.Inventory, error) {
	s.calls++
	s.gotProductID = productID
	if s.err != nil {
		return nil, s.err
	}
	return s.found, nil
}

func TestInventoryHandler_GetInventory(t *testing.T) {
	found := &inventory.Inventory{
		ID:               7,
		ProductID:        42,
		PhysicalQuantity: 12,
		ReservedQuantity: 5,
		CreatedAt:        time.Date(2026, time.November, 10, 9, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, time.November, 10, 10, 0, 0, 0, time.UTC),
	}
	wantJSON := map[string]string{
		"id":                 "7",
		"product_id":         "42",
		"physical_quantity":  "12",
		"reserved_quantity":  "5",
		"available_quantity": "7",
		"created_at":         `"2026-11-10T09:00:00Z"`,
		"updated_at":         `"2026-11-10T10:00:00Z"`,
	}

	tests := []struct {
		name       string
		productID  string
		serviceErr error
		wantStatus int
		wantError  string
		wantCalls  int
	}{
		{
			name:       "successful retrieval",
			productID:  "42",
			wantStatus: http.StatusOK,
			wantCalls:  1,
		},
		{
			name:       "non-numeric product ID",
			productID:  "invalid",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid product ID",
		},
		{
			name:       "zero product ID",
			productID:  "0",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid product ID",
		},
		{
			name:       "negative product ID",
			productID:  "-42",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid product ID",
		},
		{
			name:       "inventory not found",
			productID:  "42",
			serviceErr: fmt.Errorf("lookup: %w", inventory.ErrNotFound),
			wantStatus: http.StatusNotFound,
			wantError:  "inventory not found",
			wantCalls:  1,
		},
		{
			name:       "unexpected service error",
			productID:  "42",
			serviceErr: errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal Server Error",
			wantCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &getInventoryServiceStub{found: found, err: tt.serviceErr}
			handler := NewInventoryHandler(service)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.SetPathValue("id", tt.productID)
			recorder := httptest.NewRecorder()

			handler.GetInventory(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body = %q", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if service.calls != tt.wantCalls {
				t.Errorf("GetInventory calls = %d, want %d", service.calls, tt.wantCalls)
			}
			if tt.wantCalls > 0 && service.gotProductID != 42 {
				t.Errorf("GetInventory called with product ID %d, want 42", service.gotProductID)
			}
			if tt.wantError != "" {
				wantBody := fmt.Sprintf("{\"error\":%q}\n", tt.wantError)
				if got := recorder.Body.String(); got != wantBody {
					t.Errorf("response body = %q, want %q", got, wantBody)
				}
				return
			}

			var response map[string]json.RawMessage
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode inventory response: %v", err)
			}
			for field, want := range wantJSON {
				if got := string(response[field]); got != want {
					t.Errorf("response field %q = %q, want %q", field, got, want)
				}
			}
		})
	}
}

type adjustPhysicalQuantityServiceStub struct {
	InventoryService
	err          error
	calls        int
	gotProductID int64
	gotDelta     int32
}

func (s *adjustPhysicalQuantityServiceStub) AdjustPhysicalQuantity(_ context.Context, productID int64, delta int32) error {
	s.calls++
	s.gotProductID = productID
	s.gotDelta = delta
	return s.err
}

func TestInventoryHandler_AdjustPhysicalQuantity(t *testing.T) {
	const validBody = `{"delta":-3}`

	tests := []struct {
		name       string
		productID  string
		body       string
		serviceErr error
		wantStatus int
		wantError  string
		wantCalls  int
		wantDelta  int32
	}{
		{
			name:       "successful decrease",
			productID:  "42",
			body:       validBody,
			wantStatus: http.StatusNoContent,
			wantCalls:  1,
			wantDelta:  -3,
		},
		{
			name:       "successful increase",
			productID:  "42",
			body:       `{"delta":3}`,
			wantStatus: http.StatusNoContent,
			wantCalls:  1,
			wantDelta:  3,
		},
		{
			name:       "invalid product ID",
			productID:  "invalid",
			body:       validBody,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid product ID",
		},
		{
			name:       "malformed JSON",
			productID:  "42",
			body:       `{"delta":`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid JSON request body",
		},
		{
			name:       "unknown JSON field",
			productID:  "42",
			body:       `{"delta":-3,"extra":true}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid JSON request body",
		},
		{
			name:       "multiple JSON values",
			productID:  "42",
			body:       validBody + " {}",
			wantStatus: http.StatusBadRequest,
			wantError:  "request body must contain a single JSON value",
		},
		{
			name:       "oversized body",
			productID:  "42",
			body:       validBody + strings.Repeat(" ", maxRequestBodySize),
			wantStatus: http.StatusRequestEntityTooLarge,
			wantError:  "request body too large",
		},
		{
			name:       "zero delta",
			productID:  "42",
			body:       `{"delta":0}`,
			serviceErr: inventory.ErrInvalidAdjustment,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid inventory adjustment",
			wantCalls:  1,
		},
		{
			name:       "insufficient physical stock",
			productID:  "42",
			body:       validBody,
			serviceErr: fmt.Errorf("adjust: %w", inventory.ErrInsufficientPhysicalStock),
			wantStatus: http.StatusConflict,
			wantError:  "insufficient physical stock",
			wantCalls:  1,
			wantDelta:  -3,
		},
		{
			name:       "inventory not found",
			productID:  "42",
			body:       validBody,
			serviceErr: fmt.Errorf("adjust: %w", inventory.ErrNotFound),
			wantStatus: http.StatusNotFound,
			wantError:  "inventory not found",
			wantCalls:  1,
			wantDelta:  -3,
		},
		{
			name:       "unexpected service error",
			productID:  "42",
			body:       validBody,
			serviceErr: errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Internal Server Error",
			wantCalls:  1,
			wantDelta:  -3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &adjustPhysicalQuantityServiceStub{err: tt.serviceErr}
			handler := NewInventoryHandler(service)
			req := httptest.NewRequest(http.MethodPatch, "/", strings.NewReader(tt.body))
			req.SetPathValue("id", tt.productID)
			recorder := httptest.NewRecorder()

			handler.AdjustPhysicalQuantity(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body = %q", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			wantBody := ""
			if tt.wantError != "" {
				wantBody = fmt.Sprintf("{\"error\":%q}\n", tt.wantError)
			}
			if got := recorder.Body.String(); got != wantBody {
				t.Errorf("response body = %q, want %q", got, wantBody)
			}
			if service.calls != tt.wantCalls {
				t.Fatalf("AdjustPhysicalQuantity calls = %d, want %d", service.calls, tt.wantCalls)
			}
			if tt.wantCalls > 0 && (service.gotProductID != 42 || service.gotDelta != tt.wantDelta) {
				t.Errorf("AdjustPhysicalQuantity called with product ID %d and delta %d, want 42 and %d",
					service.gotProductID, service.gotDelta, tt.wantDelta)
			}
		})
	}
}

type releaseStockServiceStub struct {
	InventoryService
	err          error
	calls        int
	gotProductID int64
	gotQuantity  int32
}

func (s *releaseStockServiceStub) ReleaseStock(_ context.Context, productID int64, quantity int32) error {
	s.calls++
	s.gotProductID = productID
	s.gotQuantity = quantity
	return s.err
}

func TestInventoryHandler_ReleaseStock(t *testing.T) {
	const validBody = `{"quantity":3}`

	tests := []struct {
		name         string
		productID    string
		body         string
		serviceErr   error
		wantStatus   int
		wantError    string
		wantCalls    int
		wantQuantity int32
	}{
		{
			name:         "successful release",
			productID:    "42",
			body:         validBody,
			wantStatus:   http.StatusNoContent,
			wantCalls:    1,
			wantQuantity: 3,
		},
		{
			name:       "non-numeric product ID",
			productID:  "invalid",
			body:       validBody,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid product ID",
		},
		{
			name:       "zero product ID",
			productID:  "0",
			body:       validBody,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid product ID",
		},
		{
			name:       "negative product ID",
			productID:  "-42",
			body:       validBody,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid product ID",
		},
		{
			name:       "invalid JSON",
			productID:  "42",
			body:       `{"quantity":`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid JSON request body",
		},
		{
			name:       "unknown JSON field",
			productID:  "42",
			body:       `{"quantity":3,"extra":true}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid JSON request body",
		},
		{
			name:       "multiple JSON values",
			productID:  "42",
			body:       validBody + " {}",
			wantStatus: http.StatusBadRequest,
			wantError:  "request body must contain a single JSON value",
		},
		{
			name:       "empty body",
			productID:  "42",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid JSON request body",
		},
		{
			name:       "oversized body",
			productID:  "42",
			body:       validBody + strings.Repeat(" ", maxRequestBodySize),
			wantStatus: http.StatusRequestEntityTooLarge,
			wantError:  "request body too large",
		},
		{
			name:       "invalid reservation quantity",
			productID:  "42",
			body:       `{"quantity":0}`,
			serviceErr: inventory.ErrInvalidReservation,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid reservation",
			wantCalls:  1,
		},
		{
			name:         "insufficient reserved stock",
			productID:    "42",
			body:         validBody,
			serviceErr:   fmt.Errorf("release stock: %w", inventory.ErrInsufficientReservedStock),
			wantStatus:   http.StatusConflict,
			wantError:    "insufficient reserved stock",
			wantCalls:    1,
			wantQuantity: 3,
		},
		{
			name:         "inventory not found",
			productID:    "42",
			body:         validBody,
			serviceErr:   fmt.Errorf("release stock: %w", inventory.ErrNotFound),
			wantStatus:   http.StatusNotFound,
			wantError:    "inventory not found",
			wantCalls:    1,
			wantQuantity: 3,
		},
		{
			name:         "unexpected service error",
			productID:    "42",
			body:         validBody,
			serviceErr:   errors.New("database unavailable"),
			wantStatus:   http.StatusInternalServerError,
			wantError:    "Internal Server Error",
			wantCalls:    1,
			wantQuantity: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &releaseStockServiceStub{err: tt.serviceErr}
			handler := NewInventoryHandler(service)
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.SetPathValue("id", tt.productID)
			recorder := httptest.NewRecorder()

			handler.ReleaseStock(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body = %q", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			wantBody := ""
			if tt.wantError != "" {
				wantBody = fmt.Sprintf("{\"error\":%q}\n", tt.wantError)
			}
			if got := recorder.Body.String(); got != wantBody {
				t.Errorf("response body = %q, want %q", got, wantBody)
			}
			if service.calls != tt.wantCalls {
				t.Fatalf("ReleaseStock calls = %d, want %d", service.calls, tt.wantCalls)
			}
			if tt.wantCalls == 0 {
				return
			}
			if service.gotProductID != 42 || service.gotQuantity != tt.wantQuantity {
				t.Errorf("ReleaseStock called with product ID %d and quantity %d, want 42 and %d",
					service.gotProductID, service.gotQuantity, tt.wantQuantity)
			}
		})
	}
}

type reserveStockServiceStub struct {
	InventoryService
	err          error
	calls        int
	gotProductID int64
	gotQuantity  int32
}

func (s *reserveStockServiceStub) ReserveStock(_ context.Context, productID int64, quantity int32) error {
	s.calls++
	s.gotProductID = productID
	s.gotQuantity = quantity
	return s.err
}

func TestInventoryHandler_ReserveStock(t *testing.T) {
	const validBody = `{"quantity":3}`

	tests := []struct {
		name         string
		productID    string
		body         string
		serviceErr   error
		wantStatus   int
		wantError    string
		wantCalls    int
		wantQuantity int32
	}{
		{
			name:         "successful reservation",
			productID:    "42",
			body:         validBody,
			wantStatus:   http.StatusNoContent,
			wantCalls:    1,
			wantQuantity: 3,
		},
		{
			name:       "non-numeric product ID",
			productID:  "invalid",
			body:       validBody,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid product ID",
		},
		{
			name:       "non-positive product ID",
			productID:  "0",
			body:       validBody,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid product ID",
		},
		{
			name:       "negative product ID",
			productID:  "-42",
			body:       validBody,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid product ID",
		},
		{
			name:       "invalid JSON",
			productID:  "42",
			body:       `{"quantity":`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid JSON request body",
		},
		{
			name:       "unknown JSON field",
			productID:  "42",
			body:       `{"quantity":3,"extra":true}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid JSON request body",
		},
		{
			name:       "multiple JSON values",
			productID:  "42",
			body:       validBody + " {}",
			wantStatus: http.StatusBadRequest,
			wantError:  "request body must contain a single JSON value",
		},
		{
			name:       "empty body",
			productID:  "42",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid JSON request body",
		},
		{
			name:       "oversized body",
			productID:  "42",
			body:       validBody + strings.Repeat(" ", maxRequestBodySize),
			wantStatus: http.StatusRequestEntityTooLarge,
			wantError:  "request body too large",
		},
		{
			name:       "invalid reservation",
			productID:  "42",
			body:       `{"quantity":0}`,
			serviceErr: inventory.ErrInvalidReservation,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid reservation",
			wantCalls:  1,
		},
		{
			name:         "insufficient stock",
			productID:    "42",
			body:         validBody,
			serviceErr:   fmt.Errorf("reserve stock: %w", inventory.ErrInsufficientStock),
			wantStatus:   http.StatusConflict,
			wantError:    "insufficient stock",
			wantCalls:    1,
			wantQuantity: 3,
		},
		{
			name:         "inventory not found",
			productID:    "42",
			body:         validBody,
			serviceErr:   fmt.Errorf("reserve stock: %w", inventory.ErrNotFound),
			wantStatus:   http.StatusNotFound,
			wantError:    "inventory not found",
			wantCalls:    1,
			wantQuantity: 3,
		},
		{
			name:         "unexpected service error",
			productID:    "42",
			body:         validBody,
			serviceErr:   errors.New("database unavailable"),
			wantStatus:   http.StatusInternalServerError,
			wantError:    "Internal Server Error",
			wantCalls:    1,
			wantQuantity: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &reserveStockServiceStub{err: tt.serviceErr}
			handler := NewInventoryHandler(service)
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.SetPathValue("id", tt.productID)
			recorder := httptest.NewRecorder()

			handler.ReserveStock(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body = %q", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			wantBody := ""
			if tt.wantError != "" {
				wantBody = fmt.Sprintf("{\"error\":%q}\n", tt.wantError)
			}
			if got := recorder.Body.String(); got != wantBody {
				t.Errorf("response body = %q, want %q", got, wantBody)
			}
			if service.calls != tt.wantCalls {
				t.Fatalf("ReserveStock calls = %d, want %d", service.calls, tt.wantCalls)
			}
			if tt.wantCalls == 0 {
				return
			}
			if service.gotProductID != 42 || service.gotQuantity != tt.wantQuantity {
				t.Errorf("ReserveStock called with product ID %d and quantity %d, want 42 and %d",
					service.gotProductID, service.gotQuantity, tt.wantQuantity)
			}
		})
	}
}
