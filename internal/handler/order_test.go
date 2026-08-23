package handler

import (
	"context"
	"delivery-api/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

var _ OrderService = (*fakeService)(nil)

type fakeService struct {
	order       model.Order
	err         error
	createCall  bool
	createModel model.Order
}

func (f *fakeService) GetOrders(ctx context.Context) ([]model.Order, error) {
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

func TestGetOrderByIDInvalidID(t *testing.T) {
	healthHandler := NewHealthHandler(nil)
	orderHandler := NewOrderHandler(&fakeService{})
	router := NewRouter(orderHandler, healthHandler)
	req := httptest.NewRequest("GET", "/orders/abc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Получили: %d; Ожидали: %d", rec.Code, http.StatusBadRequest)
	}
}
