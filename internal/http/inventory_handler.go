package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/MohanRaj-1/go-order-management/internal/inventory"
)

type InventoryService interface {
	GetInventory(ctx context.Context, productID int64) (*inventory.Inventory, error)
	AdjustPhysicalQuantity(ctx context.Context, productID int64, delta int32) error
	ReserveStock(ctx context.Context, productID int64, quantity int32) error
	ReleaseStock(ctx context.Context, productID int64, quantity int32) error
}

type InventoryHandler struct {
	service InventoryService
}

func NewInventoryHandler(service InventoryService) *InventoryHandler {
	return &InventoryHandler{
		service: service,
	}
}

type adjustPhysicalQuantityRequest struct {
	Delta int32 `json:"delta"`
}

type stockQuantityRequest struct {
	Quantity int32 `json:"quantity"`
}

type inventoryResponse struct {
	ID                int64     `json:"id"`
	ProductID         int64     `json:"product_id"`
	PhysicalQuantity  int32     `json:"physical_quantity"`
	ReservedQuantity  int32     `json:"reserved_quantity"`
	AvailableQuantity int32     `json:"available_quantity"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func toInventoryResponse(i *inventory.Inventory) inventoryResponse {
	return inventoryResponse{
		ID:                i.ID,
		ProductID:         i.ProductID,
		PhysicalQuantity:  i.PhysicalQuantity,
		ReservedQuantity:  i.ReservedQuantity,
		AvailableQuantity: i.AvailableQuantity(),
		CreatedAt:         i.CreatedAt,
		UpdatedAt:         i.UpdatedAt,
	}
}

func (h *InventoryHandler) GetInventory(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || productID <= 0 {
		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "invalid product ID"})
		return
	}

	found, err := h.service.GetInventory(r.Context(), productID)
	if err != nil {
		writeInventoryError(w, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, toInventoryResponse(found)); err != nil {
		// The response may already have been partially written.
		return
	}
}

func writeInventoryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, inventory.ErrInvalidAdjustment):
		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "invalid inventory adjustment"})

	case errors.Is(err, inventory.ErrInvalidReservation):
		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "invalid reservation"})

	case errors.Is(err, inventory.ErrInvalidProductID):
		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "invalid product ID"})

	case errors.Is(err, inventory.ErrNotFound):
		writeJSON(w, http.StatusNotFound,
			errorResponse{Error: "inventory not found"})

	case errors.Is(err, inventory.ErrInsufficientPhysicalStock):
		writeJSON(w, http.StatusConflict,
			errorResponse{Error: "insufficient physical stock"})

	case errors.Is(err, inventory.ErrInsufficientStock):
		writeJSON(w, http.StatusConflict,
			errorResponse{Error: "insufficient stock"})

	case errors.Is(err, inventory.ErrInsufficientReservedStock):
		writeJSON(w, http.StatusConflict,
			errorResponse{Error: "insufficient reserved stock"})

	default:
		writeJSON(w, http.StatusInternalServerError,
			errorResponse{Error: http.StatusText(http.StatusInternalServerError)})
	}
}

func (h *InventoryHandler) AdjustPhysicalQuantity(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || productID <= 0 {
		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "invalid product ID"})
		return
	}

	var req adjustPhysicalQuantityRequest
	if !decodeInventoryRequest(w, r, &req) {
		return
	}

	if err := h.service.AdjustPhysicalQuantity(r.Context(), productID, req.Delta); err != nil {
		writeInventoryError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *InventoryHandler) ReserveStock(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || productID <= 0 {
		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "invalid product ID"})
		return
	}

	var req stockQuantityRequest
	if !decodeInventoryRequest(w, r, &req) {
		return
	}

	if err := h.service.ReserveStock(r.Context(), productID, req.Quantity); err != nil {
		writeInventoryError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *InventoryHandler) ReleaseStock(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || productID <= 0 {
		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "invalid product ID"})
		return
	}

	var req stockQuantityRequest
	if !decodeInventoryRequest(w, r, &req) {
		return
	}

	if err := h.service.ReleaseStock(r.Context(), productID, req.Quantity); err != nil {
		writeInventoryError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeInventoryRequest(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeJSON(w, http.StatusRequestEntityTooLarge,
				errorResponse{Error: "request body too large"})
			return false
		}
		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "invalid JSON request body"})
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeJSON(w, http.StatusRequestEntityTooLarge,
				errorResponse{Error: "request body too large"})
			return false
		}
		writeJSON(w, http.StatusBadRequest,
			errorResponse{Error: "request body must contain a single JSON value"})
		return false
	}

	return true
}
