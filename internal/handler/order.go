package handler

import (
	"context"
	"delivery-api/internal/model"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
	"time"
)

type OrderService interface {
	GetOrders(ctx context.Context, limit, offset int) ([]model.Order, error)
	GetOrderByID(ctx context.Context, id int) (model.Order, error)
	CreateOrder(ctx context.Context, o model.Order) (model.Order, error)
	UpdateOrder(ctx context.Context, id int, o model.Order) (model.Order, error)
	DeleteOrder(ctx context.Context, id int) error
}

type OrderHandler struct {
	service OrderService
	timeout time.Duration
}

func NewOrderHandler(service OrderService, timeout time.Duration) *OrderHandler {
	return &OrderHandler{service: service, timeout: timeout}
}

func queryInt(r *http.Request, key string) (int, error) {
	str := r.URL.Query().Get(key)
	if str == "" {
		return 0, nil
	}
	num, err := strconv.Atoi(str)
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	limit, err := queryInt(r, "limit")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid limit")
		return
	}

	offset, err := queryInt(r, "offset")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid offset")
		return
	}

	orders, err := h.service.GetOrders(ctx, limit, offset)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

func (h *OrderHandler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	id := chi.URLParam(r, "id")
	intID, err := strconv.Atoi(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	order, err := h.service.GetOrderByID(ctx, intID)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	var o model.Order
	err := json.NewDecoder(r.Body).Decode(&o)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	order, err := h.service.CreateOrder(ctx, o)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

func (h *OrderHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	id := chi.URLParam(r, "id")
	intID, err := strconv.Atoi(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	var o model.Order
	err = json.NewDecoder(r.Body).Decode(&o)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.service.UpdateOrder(ctx, intID, o)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	id := chi.URLParam(r, "id")
	intID, err := strconv.Atoi(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	err = h.service.DeleteOrder(ctx, intID)
	if err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
