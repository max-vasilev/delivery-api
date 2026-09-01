package main

import (
	"context"
	"database/sql"
	"delivery-api/internal/config"
	"delivery-api/internal/handler"
	"delivery-api/internal/repository"
	"delivery-api/internal/service"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	l := slog.New(jsonHandler)
	slog.SetDefault(l)

	cfg, err := config.Load()
	if err != nil {
		l.Error("config load failed", slog.Any("error", err))
		os.Exit(1)
	}

	db, err := sql.Open("pgx", cfg.DB.URL)
	if err != nil {
		l.Error("database open failed", slog.Any("error", err))
		os.Exit(1)
	}

	defer func() { _ = db.Close() }()
	err = db.Ping()
	if err != nil {
		l.Error("database ping failed", slog.Any("error", err))
		os.Exit(1)
	}
	l.Info("database connected")

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(time.Minute * 5)
	db.SetConnMaxIdleTime(time.Minute * 2)

	orderRepo := repository.NewOrderRepository(db)
	orderService := service.NewOrderService(orderRepo)
	orderHandler := handler.NewOrderHandler(orderService, cfg.Server.RequestTimeout)
	healthHandler := handler.NewHealthHandler(db)

	router := handler.NewRouter(orderHandler, healthHandler, l)

	server := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	l.Info("starting server", slog.String("addr", cfg.Server.Addr()))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		l.Error("server error", slog.Any("error", err))
	case <-quit:
		l.Info("shutting down server")

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			l.Error("shutdown timeout exceeded", slog.Any("error", err))
		} else {
			l.Info("server stopped")
		}
	}
}
