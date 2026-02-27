// Package http provides HTTP handlers for the orders module.
package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rai/clean-modularmonolith-go/modules/orders/application/commands"
	"github.com/rai/clean-modularmonolith-go/modules/orders/application/queries"
	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
)

type Handler struct {
	createOrder *commands.CreateOrderHandler
	addItem     *commands.AddItemHandler
	removeItem  *commands.RemoveItemHandler
	submitOrder *commands.SubmitOrderHandler
	cancelOrder *commands.CancelOrderHandler
	getOrder    *queries.GetOrderHandler
	listOrders  *queries.ListUserOrdersHandler
}

// RegisterRoutes registers the orders module routes to the given mux.
func RegisterRoutes(
	mux *http.ServeMux,
	createOrder *commands.CreateOrderHandler,
	addItem *commands.AddItemHandler,
	removeItem *commands.RemoveItemHandler,
	submitOrder *commands.SubmitOrderHandler,
	cancelOrder *commands.CancelOrderHandler,
	getOrder *queries.GetOrderHandler,
	listOrders *queries.ListUserOrdersHandler,
) {
	h := &Handler{
		createOrder: createOrder,
		addItem:     addItem,
		removeItem:  removeItem,
		submitOrder: submitOrder,
		cancelOrder: cancelOrder,
		getOrder:    getOrder,
		listOrders:  listOrders,
	}

	mux.HandleFunc("POST /orders", h.handleCreateOrder)
	mux.HandleFunc("GET /orders/{id}", h.handleGetOrder)
	mux.HandleFunc("POST /orders/{id}/items", h.handleAddItem)
	mux.HandleFunc("DELETE /orders/{id}/items/{productId}", h.handleRemoveItem)
	mux.HandleFunc("POST /orders/{id}/submit", h.handleSubmitOrder)
	mux.HandleFunc("POST /orders/{id}/cancel", h.handleCancelOrder)
	mux.HandleFunc("GET /users/{userId}/orders", h.handleListUserOrders)
}

// Request/Response DTOs

type createOrderRequest struct {
	UserID string `json:"user_id"`
}

type createOrderResponse struct {
	ID string `json:"id"`
}

type addItemRequest struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
	Currency    string `json:"currency"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Handlers

func (h *Handler) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cmd := commands.CreateOrderCommand{UserID: req.UserID}
	id, err := h.createOrder.Handle(r.Context(), cmd)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, createOrderResponse{ID: id})
}

func (h *Handler) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "order ID is required")
		return
	}

	query := queries.GetOrderQuery{OrderID: id}
	order, err := h.getOrder.Handle(r.Context(), query)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func (h *Handler) handleAddItem(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	if orderID == "" {
		writeError(w, http.StatusBadRequest, "order ID is required")
		return
	}

	var req addItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cmd := commands.AddItemCommand{
		OrderID:     orderID,
		ProductID:   req.ProductID,
		ProductName: req.ProductName,
		Quantity:    req.Quantity,
		UnitPrice:   req.UnitPrice,
		Currency:    req.Currency,
	}

	if err := h.addItem.Handle(r.Context(), cmd); err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleRemoveItem(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	productID := r.PathValue("productId")

	cmd := commands.RemoveItemCommand{
		OrderID:   orderID,
		ProductID: productID,
	}

	if err := h.removeItem.Handle(r.Context(), cmd); err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleSubmitOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")

	cmd := commands.SubmitOrderCommand{OrderID: orderID}
	if err := h.submitOrder.Handle(r.Context(), cmd); err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")

	cmd := commands.CancelOrderCommand{OrderID: orderID}
	order, err := h.cancelOrder.Handle(r.Context(), cmd)
	if err != nil {
		handleError(w, err)
		return
	}
	fmt.Println(order.ID()) // TODO

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleListUserOrders(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userId")
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	query := queries.ListUserOrdersQuery{
		UserID: userID,
		Offset: offset,
		Limit:  limit,
	}

	result, err := h.listOrders.Handle(r.Context(), query)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Helper functions

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrOrderNotDraft):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrOrderEmpty):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrItemNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidQuantity):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
