package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mhmddaud/transjakarta-fleet/internal/config"
	"github.com/mhmddaud/transjakarta-fleet/internal/database"
	"github.com/mhmddaud/transjakarta-fleet/internal/handler"
	"github.com/mhmddaud/transjakarta-fleet/internal/rabbitmq"
	"github.com/mhmddaud/transjakarta-fleet/internal/repository"
	"github.com/mhmddaud/transjakarta-fleet/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	db, err := database.NewPostgresPool(dbCtx, cfg)
	if err != nil {
		log.Fatalf("connect PostgreSQL: %v", err)
	}
	defer db.Close()

	publisher, err := rabbitmq.NewPublisher(
		cfg.RabbitMQURL,
		cfg.RabbitMQExchange,
		cfg.RabbitMQQueue,
	)
	if err != nil {
		log.Fatalf("connect RabbitMQ: %v", err)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			log.Printf("close RabbitMQ publisher: %v", err)
		}
	}()

	locationRepo := repository.NewLocationRepository(db)

	geofence := service.NewGeofenceService(
		cfg.GeofenceLatitude,
		cfg.GeofenceLongitude,
		cfg.GeofenceRadius,
	)

	locationSvc := service.NewLocationService(
		locationRepo,
		geofence,
		publisher,
	)

	locationHandler := handler.NewLocationHandler(locationSvc)

	mux := http.NewServeMux()
	mux.Handle("/api/v1/locations", locationHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("HTTP server listening on :%s", cfg.AppPort)
		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown: %v", err)
	}
}
