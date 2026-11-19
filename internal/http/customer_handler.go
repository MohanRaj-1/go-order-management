package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/MohanRaj-1/go-order-management/internal/customer"
)

const maxRequestBodySize = 1 << 20 // 1 MB

type CustomerService interface {
	CreateCustomer(ctx context.Context, name, email string) (*customer.Customer, error)
	GetCustomerByID(ctx context.Context, id int64) (*customer.Customer, error)
}

func (h *CustomerHandler) GetCustomerByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeCustomerError(w, customer.ErrInvalidCustomerID)
		return
	}

	c, err := h.service.GetCustomerByID(r.Context(), id)
	if err != nil {
		writeCustomerError(w, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, toCustomerResponse(c)); err != nil {
		// Response may already have been partially written.
		return
	}
}

type CustomerHandler struct {
	service CustomerService
}

func NewCustomerHandler(service CustomerService) *CustomerHandler {
	return &CustomerHandler{
		service: service,
	}
}

type createCustomerRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type customerResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func toCustomerResponse(c *customer.Customer) customerResponse {
	return customerResponse{
		ID:        c.ID,
		Name:      c.Name,
		Email:     c.Email,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	return json.NewEncoder(w).Encode(data)
}

func writeCustomerError(w http.ResponseWriter, err error) {
	var statusCode int
	var message string

	switch {
	case errors.Is(err, customer.ErrInvalidName):
		statusCode = http.StatusBadRequest
		message = "invalid customer name"

	case errors.Is(err, customer.ErrInvalidEmail):
		statusCode = http.StatusBadRequest
		message = "invalid customer email"

	case errors.Is(err, customer.ErrInvalidCustomerID):
		statusCode = http.StatusBadRequest
		message = "invalid customer ID"

	case errors.Is(err, customer.ErrNotFound):
		statusCode = http.StatusNotFound
		message = "customer not found"

	default:
		statusCode = http.StatusInternalServerError
		message = http.StatusText(statusCode)
	}

	if writeErr := writeJSON(w, statusCode, errorResponse{Error: message}); writeErr != nil {
		// The response may already have been partially written.
		// There is no reliable alternative response to send here.
	}
}

func (h *CustomerHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var req createCustomerRequest

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError

		if errors.As(err, &maxBytesErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{
				Error: "request body too large",
			})
			return
		}

		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "invalid request body",
		})
		return
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		var maxBytesErr *http.MaxBytesError

		if errors.As(err, &maxBytesErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{
				Error: "request body too large",
			})
			return
		}

		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "request body must contain a single JSON object",
		})
		return
	}

	c, err := h.service.CreateCustomer(r.Context(), req.Name, req.Email)
	if err != nil {
		writeCustomerError(w, err)
		return
	}

	if err := writeJSON(w, http.StatusCreated, toCustomerResponse(c)); err != nil {
		// Response may already have been partially written.
		return
	}
}
