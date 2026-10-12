package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/mhmddaud/transjakarta-fleet/internal/service"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	mu       sync.Mutex
	conn     *amqp.Connection
	channel  *amqp.Channel
	url      string
	exchange string
	queue    string
}

func NewPublisher(url, exchange, queue string) (*Publisher, error) {
	if url == "" || exchange == "" || queue == "" {
		return nil, errors.New(
			"RabbitMQ URL, exchange, and queue are required",
		)
	}

	p := &Publisher{
		url:      url,
		exchange: exchange,
		queue:    queue,
	}

	p.mu.Lock()
	err := p.connectLocked()
	p.mu.Unlock()

	if err != nil {
		return nil, err
	}

	return p, nil
}

// connectLocked membuat koneksi, channel, dan topology RabbitMQ.
// Pemanggil wajib memegang mutex p.mu.
func (p *Publisher) connectLocked() error {
	if p.channel != nil {
		_ = p.channel.Close()
		p.channel = nil
	}
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}

	conn, err := amqp.Dial(p.url)
	if err != nil {
		return fmt.Errorf("connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open RabbitMQ channel: %w", err)
	}

	if err := declareTopology(channel, p.exchange, p.queue); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return fmt.Errorf("declare RabbitMQ topology: %w", err)
	}

	for _, routingKey := range []string{
		"vehicle.entered_geofence",
		"vehicle.exited_geofence",
	} {
		if err := channel.QueueBind(
			p.queue,
			routingKey,
			p.exchange,
			false,
			nil,
		); err != nil {
			_ = channel.Close()
			_ = conn.Close()
			return fmt.Errorf("bind queue: %w", err)
		}
	}

	p.conn = conn
	p.channel = channel

	return nil
}

func (p *Publisher) publishLocked(
	ctx context.Context,
	event service.GeofenceEvent,
) error {
	if p.channel == nil ||
		p.conn == nil ||
		p.conn.IsClosed() ||
		p.channel.IsClosed() {
		if err := p.connectLocked(); err != nil {
			return err
		}
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

func (p *Publisher) PublishEvent(
	ctx context.Context,
	event service.GeofenceEvent,
) error {
	if p == nil {
		return errors.New("RabbitMQ publisher is not initialized")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	err := p.publishLocked(ctx, event)
	if err == nil {
		return nil
	}

	// Jika publish gagal, buka koneksi/channel baru dan coba sekali lagi.
	if reconnectErr := p.connectLocked(); reconnectErr != nil {
		return fmt.Errorf(
			"publish failed: %v; reconnect failed: %w",
			err,
			reconnectErr,
		)
	}

	if retryErr := p.publishLocked(ctx, event); retryErr != nil {
		return fmt.Errorf(
			"publish failed after reconnect: %w",
			retryErr,
		)
	}

	return nil
}

func (p *Publisher) Close() error {
	if p == nil {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error

	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			firstErr = err
		}
		p.channel = nil
	}

	if p.conn != nil {
		if err := p.conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		p.conn = nil
	}

	return firstErr
}
