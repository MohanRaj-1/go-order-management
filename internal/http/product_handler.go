package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/MohanRaj-1/go-order-management/internal/product"
)

type ProductService interface {
	CreateProduct(
		ctx context.Context,
		sku string,
		name string,
		description string,
		price int64,
	) (*product.Product, error)

	GetProductByID(ctx context.Context, id int64) (*product.Product, error)
}

type ProductHandler struct {
	service ProductService
}

func NewProductHandler(service ProductService) *ProductHandler {
	return &ProductHandler{
		service: service,
	}
}

type createProductRequest struct {
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
}

type productResponse struct {
	ID          int64                 `json:"id"`
	SKU         string                `json:"sku"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Price       int64                 `json:"price"`
	Status      product.ProductStatus `json:"status"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

func toProductResponse(p *product.Product) productResponse {
	return productResponse{
		ID:          p.ID,
		SKU:         p.SKU,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var req createProductRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeJSON(w, http.StatusRequestEntityTooLarge,
				errorResponse{Error: "request body too large"})
			return
		}

		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "invalid JSON request body"})
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeJSON(w, http.StatusRequestEntityTooLarge,
				errorResponse{Error: "request body too large"})
			return
		}

		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "request body must contain a single JSON value"})
		return
	}

	created, err := h.service.CreateProduct(
		r.Context(),
		req.SKU,
		req.Name,
		req.Description,
		req.Price,
	)
	if err != nil {
		writeProductError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toProductResponse(created))
}

func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "invalid product ID"})
		return
	}

	found, err := h.service.GetProductByID(r.Context(), id)
	if err != nil {
		writeProductError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toProductResponse(found))
}

func writeProductError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, product.ErrInvalidSKU):
		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "invalid product SKU"})

	case errors.Is(err, product.ErrInvalidName):
		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "invalid product name"})

	case errors.Is(err, product.ErrInvalidPrice):
		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "invalid product price"})

	case errors.Is(err, product.ErrNotFound):
		writeJSON(w, http.StatusNotFound,
			errorResponse{Error: "product not found"})

	default:
		writeJSON(w, http.StatusInternalServerError,
			errorResponse{Error: http.StatusText(http.StatusInternalServerError)})
	}
}
