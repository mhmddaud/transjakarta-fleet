package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mhmddaud/transjakarta-fleet/internal/config"
	"github.com/mhmddaud/transjakarta-fleet/internal/rabbitmq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	consumer, err := rabbitmq.NewConsumerWithExchange(
		cfg.RabbitMQURL,
		cfg.RabbitMQExchange,
		cfg.RabbitMQQueue,
	)
	if err != nil {
		log.Fatalf("failed to initialize RabbitMQ consumer: %v", err)
	}
	defer consumer.Close()

	handler := func(
		ctx context.Context,
		event rabbitmq.GeofenceEvent,
	) error {
		log.Printf(
			"Geofence notification: vehicle=%s event=%s lat=%f lon=%f timestamp=%d",
			event.VehicleID,
			event.EventType,
			event.Latitude,
			event.Longitude,
			event.Timestamp,
		)

		// Tambahkan business logic notifikasi di sini.
		return nil
	}

	log.Println("Starting Transjakarta Fleet consumer...")

	if err := consumer.Run(ctx, handler); err != nil {
		log.Printf("Consumer stopped: %v", err)
	}
}
