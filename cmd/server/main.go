package main

import (
	"context"
	"database/sql"
	"delivery-api/internal/config"
	"delivery-api/internal/handler"
	"delivery-api/internal/repository"
	"delivery-api/internal/service"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("pgx", cfg.DB.URL)
	if err != nil {
		log.Fatal(err)
	}

	defer func() { _ = db.Close() }()
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Success db connection")

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(time.Minute * 5)
	db.SetConnMaxIdleTime(time.Minute * 2)

	orderRepo := repository.NewOrderRepository(db)
	orderService := service.NewOrderService(orderRepo)
	orderHandler := handler.NewOrderHandler(orderService, cfg.Server.RequestTimeout)
	healthHandler := handler.NewHealthHandler(db)

	router := handler.NewRouter(orderHandler, healthHandler)

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
	log.Printf("Starting server on %s", cfg.Server.Addr())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Println(err)
	case <-quit:
		log.Println("Останавливаю сервер...")

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Println("Сервер перестал ждать и был остановлен")
		} else {
			log.Println("Сервер успешно остановлен")
		}
	}
}
