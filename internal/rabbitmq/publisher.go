package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mhmddaud/transjakarta-fleet/internal/service"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
	queue    string
}

func NewPublisher(url, exchange, queue string) (*Publisher, error) {
	if url == "" || exchange == "" || queue == "" {
		return nil, errors.New("RabbitMQ URL, exchange, and queue are required")
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

	for _, routingKey := range []string{
		"vehicle.entered_geofence",
		"vehicle.exited_geofence",
	} {
		if err := channel.QueueBind(
			queue,
			routingKey,
			exchange,
			false,
			nil,
		); err != nil {
			_ = channel.Close()
			_ = conn.Close()
			return nil, fmt.Errorf("bind queue: %w", err)
		}
	}

	return &Publisher{
		conn:     conn,
		channel:  channel,
		exchange: exchange,
		queue:    queue,
	}, nil
}

func (p *Publisher) PublishEvent(
	ctx context.Context,
	event service.GeofenceEvent,
) error {
	if p == nil || p.channel == nil {
		return errors.New("RabbitMQ publisher is not initialized")
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal geofence event: %w", err)
	}

	return p.channel.PublishWithContext(
		ctx,
		p.exchange,
		event.EventType,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

func (p *Publisher) Close() error {
	if p == nil {
		return nil
	}

	var firstErr error

	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			firstErr = err
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
