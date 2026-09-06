package handler

import (
	"context"
	"delivery-api/internal/apperror"
	"delivery-api/internal/model"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var _ OrderService = (*fakeService)(nil)

type fakeService struct {
	order       model.Order
	err         error
	createCall  bool
	createModel model.Order
}

func (f *fakeService) GetOrders(ctx context.Context, limit, offset int) ([]model.Order, error) {
	return nil, f.err
}

func (f *fakeService) GetOrderByID(ctx context.Context, id int) (model.Order, error) {
	return f.order, f.err
}

func (f *fakeService) CreateOrder(ctx context.Context, o model.Order) (model.Order, error) {
	f.createCall = true
	f.createModel = o
	return f.order, f.err
}

func (f *fakeService) UpdateOrder(ctx context.Context, id int, o model.Order) (model.Order, error) {
	return f.order, f.err
}

func (f *fakeService) DeleteOrder(ctx context.Context, id int) error {
	return f.err
}

func TestOrdersInvalidLimit(t *testing.T) {
	healthHandler := NewHealthHandler(nil)
	orderHandler := NewOrderHandler(&fakeService{}, 3*time.Second)
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(orderHandler, healthHandler, l)
	req := httptest.NewRequest("GET", "/orders?limit=abc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Получили: %d; Ожидали: %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetOrderByIDInvalidID(t *testing.T) {
	healthHandler := NewHealthHandler(nil)
	orderHandler := NewOrderHandler(&fakeService{}, 3*time.Second)
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(orderHandler, healthHandler, l)
	req := httptest.NewRequest("GET", "/orders/abc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Получили: %d; Ожидали: %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetOrderByIDNotFound(t *testing.T) {
	healthHandler := NewHealthHandler(nil)
	orderHandler := NewOrderHandler(&fakeService{err: apperror.ErrOrderNotFound}, 3*time.Second)
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(orderHandler, healthHandler, l)
	req := httptest.NewRequest("GET", "/orders/999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("Получили: %d; Ожидали: %d", rec.Code, http.StatusNotFound)
	}
}

func TestCreateOrderCorrect(t *testing.T) {
	healthHandler := NewHealthHandler(nil)
	orderHandler := NewOrderHandler(&fakeService{}, 3*time.Second)
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(orderHandler, healthHandler, l)
	req := httptest.NewRequest("POST", "/orders", strings.NewReader(`{
"address": "г. Ижевск, ул. Пушкинская, 10",
"price": 50000,
"items": [
{"name": "Пицца", "quantity": 2, "price": 30000}
]}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("Получили: %d; Ожидали: %d", rec.Code, http.StatusCreated)
	}
}

func TestCreateOrderInvalidBody(t *testing.T) {
	healthHandler := NewHealthHandler(nil)
	orderHandler := NewOrderHandler(&fakeService{}, 3*time.Second)
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := NewRouter(orderHandler, healthHandler, l)
	req := httptest.NewRequest("POST", "/orders", strings.NewReader(`{
"address": "г. Ижевск, ул. Пушкинская, 10",
"price": 50000,
`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Получили: %d; Ожидали: %d", rec.Code, http.StatusBadRequest)
	}
}
