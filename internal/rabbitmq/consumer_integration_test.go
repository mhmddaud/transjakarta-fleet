package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestConsumerRejectsFailedEventToDLQ(t *testing.T) {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		t.Skip("set RABBITMQ_URL to run RabbitMQ integration test")
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	exchange := "test.fleet.events." + suffix
	queue := "test.geofence_alerts." + suffix
	dlq := queue + ".dlq"
	dlx := exchange + ".dlx"

	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatalf("connect RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()

	// Ensure resources are removed after the test.
	defer func() {
		_, _ = ch.QueueDelete(queue, false, false, false)
		_, _ = ch.QueueDelete(dlq, false, false, false)
		_ = ch.ExchangeDelete(exchange, false, false)
		_ = ch.ExchangeDelete(dlx, false, false)
	}()

	if err := declareTopology(ch, exchange, queue); err != nil {
		t.Fatalf("declare topology: %v", err)
	}

	consumer, err := NewConsumerWithExchange(url, exchange, queue)
	if err != nil {
		t.Fatalf("create consumer: %v", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErr := make(chan error, 1)
	go func() {
		runErr <- consumer.Run(ctx, func(
			_ context.Context,
			_ GeofenceEvent,
		) error {
			return errors.New("intentional test failure")
		})
	}()

	body, err := json.Marshal(GeofenceEvent{
		VehicleID: "TEST-DLQ-001",
		EventType: "vehicle.entered_geofence",
		Latitude:  -6.2088,
		Longitude: 106.8456,
		Timestamp: time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	if err := ch.PublishWithContext(
		context.Background(),
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	); err != nil {
		t.Fatalf("publish test event: %v", err)
	}

	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		message, ok, err := ch.Get(dlq, true)
		if err != nil {
			t.Fatalf("read dead-letter queue: %v", err)
		}
		if ok {
			var event GeofenceEvent
			if err := json.Unmarshal(message.Body, &event); err != nil {
				t.Fatalf("decode dead-letter event: %v", err)
			}
			if event.VehicleID != "TEST-DLQ-001" {
				t.Fatalf("unexpected dead-letter vehicle: %s", event.VehicleID)
			}
			cancel()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	cancel()
	t.Fatal("message did not arrive in dead-letter queue within 25 seconds")
}
