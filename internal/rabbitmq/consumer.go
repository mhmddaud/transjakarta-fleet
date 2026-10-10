package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const maxRetryAttempts = 3

type GeofenceEvent struct {
	VehicleID string  `json:"vehicle_id"`
	EventType string  `json:"event_type"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}

type EventHandler func(context.Context, GeofenceEvent) error

type Consumer struct {
	conn          *amqp.Connection
	channel       *amqp.Channel
	queue         string
	exchange      string
	retryExchange string
}

func NewConsumer(url, queue string) (*Consumer, error) {
	return NewConsumerWithExchange(url, "fleet.events", queue)
}

func NewConsumerWithExchange(
	url, exchange, queue string,
) (*Consumer, error) {
	if url == "" {
		return nil, errors.New("RabbitMQ URL is required")
	}
	if exchange == "" {
		return nil, errors.New("RabbitMQ exchange is required")
	}
	if queue == "" {
		return nil, errors.New("RabbitMQ queue is required")
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open RabbitMQ channel: %w", err)
	}

	if err := declareTopology(channel, exchange, queue); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}

	if err := channel.Qos(1, 0, false); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("set consumer qos: %w", err)
	}

	return &Consumer{
		conn:          conn,
		channel:       channel,
		queue:         queue,
		exchange:      exchange,
		retryExchange: exchange + ".retry",
	}, nil
}

func (c *Consumer) Run(
	ctx context.Context,
	handler EventHandler,
) error {
	if handler == nil {
		return errors.New("event handler is required")
	}

	deliveries, err := c.channel.ConsumeWithContext(
		ctx,
		c.queue,
		"fleet-geofence-consumer",
		false, // manual acknowledgment
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume queue: %w", err)
	}

	log.Printf("RabbitMQ consumer started; queue=%s", c.queue)

	for {
		select {
		case <-ctx.Done():
			return nil

		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("rabbitmq delivery channel closed")
			}

			var event GeofenceEvent
			if err := json.Unmarshal(delivery.Body, &event); err != nil {
				log.Printf("invalid event JSON: %v", err)
				if nackErr := delivery.Nack(false, false); nackErr != nil {
					return fmt.Errorf("reject invalid message: %w", nackErr)
				}
				continue
			}

			if err := handler(ctx, event); err != nil {
				log.Printf(
					"event processing failed; vehicle=%s type=%s error=%v",
					event.VehicleID,
					event.EventType,
					err,
				)

				count := retryCount(delivery.Headers)

				if count < maxRetryAttempts {
					if retryErr := c.publishRetry(delivery, count+1); retryErr != nil {
						return fmt.Errorf("schedule retry: %w", retryErr)
					}

					if ackErr := delivery.Ack(false); ackErr != nil {
						return fmt.Errorf("ack retried event: %w", ackErr)
					}

					log.Printf(
						"event scheduled for retry; vehicle=%s attempt=%d",
						event.VehicleID,
						count+1,
					)
					continue
				}

				log.Printf(
					"maximum retries reached; sending event to DLQ; vehicle=%s",
					event.VehicleID,
				)

				if nackErr := delivery.Nack(false, false); nackErr != nil {
					return fmt.Errorf("reject exhausted event: %w", nackErr)
				}
				continue
			}

			if err := delivery.Ack(false); err != nil {
				return fmt.Errorf("ack event: %w", err)
			}

			log.Printf(
				"event processed; vehicle=%s type=%s timestamp=%d",
				event.VehicleID,
				event.EventType,
				event.Timestamp,
			)
		}
	}
}

func (c *Consumer) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func retryCount(headers amqp.Table) int32 {
	if headers == nil {
		return 0
	}

	switch value := headers["x-retry-count"].(type) {
	case int32:
		return value
	case int64:
		return int32(value)
	case int:
		return int32(value)
	default:
		return 0
	}
}

func (c *Consumer) publishRetry(
	delivery amqp.Delivery,
	count int32,
) error {
	headers := amqp.Table{}

	for key, value := range delivery.Headers {
		headers[key] = value
	}
	headers["x-retry-count"] = count

	if err := c.channel.PublishWithContext(
		context.Background(),
		c.retryExchange,
		c.queue,
		false,
		false,
		amqp.Publishing{
			ContentType:   delivery.ContentType,
			DeliveryMode:  amqp.Persistent,
			CorrelationId: delivery.CorrelationId,
			Headers:       headers,
			Body:          delivery.Body,
		},
	); err != nil {
		return fmt.Errorf("publish retry message: %w", err)
	}

	return nil
}
