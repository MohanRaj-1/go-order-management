package main

import (
	"fmt"
	"net/http"

	httpapi "github.com/MohanRaj-1/go-order-management/internal/http"
)

func newRouter(
	customerHandler *httpapi.CustomerHandler,
	productHandler *httpapi.ProductHandler,
	inventoryHandler *httpapi.InventoryHandler,
) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)

	// Customer routes
	mux.HandleFunc("POST /customers", customerHandler.CreateCustomer)
	mux.HandleFunc("GET /customers/{id}", customerHandler.GetCustomerByID)

	// Product routes
	mux.HandleFunc("POST /products", productHandler.CreateProduct)
	mux.HandleFunc("GET /products/{id}", productHandler.GetProductByID)

	// Inventory routes
	mux.HandleFunc("GET /products/{id}/inventory", inventoryHandler.GetInventory)
	mux.HandleFunc("PATCH /products/{id}/inventory/physical-quantity", inventoryHandler.AdjustPhysicalQuantity)
	mux.HandleFunc("POST /products/{id}/inventory/reservations", inventoryHandler.ReserveStock)
	mux.HandleFunc("POST /products/{id}/inventory/reservations/release", inventoryHandler.ReleaseStock)

	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}
