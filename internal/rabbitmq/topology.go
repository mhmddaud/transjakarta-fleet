package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const retryDelayMilliseconds int32 = 5000

func declareTopology(
	channel *amqp.Channel,
	exchange string,
	queue string,
) error {
	if exchange == "" || queue == "" {
		return fmt.Errorf("RabbitMQ exchange and queue are required")
	}

	dlxExchange := exchange + ".dlx"
	dlqName := queue + ".dlq"
	retryExchange := exchange + ".retry"
	retryQueue := queue + ".retry"

	if err := channel.ExchangeDeclare(
		exchange, "topic", true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare main exchange: %w", err)
	}

	if err := channel.ExchangeDeclare(
		dlxExchange, "topic", true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare dead-letter exchange: %w", err)
	}

	if err := channel.ExchangeDeclare(
		retryExchange, "direct", true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare retry exchange: %w", err)
	}

	dlq, err := channel.QueueDeclare(
		dlqName, true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("declare dead-letter queue: %w", err)
	}

	if err := channel.QueueBind(
		dlq.Name, "#", dlxExchange, false, nil,
	); err != nil {
		return fmt.Errorf("bind dead-letter queue: %w", err)
	}

	retryArgs := amqp.Table{
		"x-message-ttl":             retryDelayMilliseconds,
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": queue,
	}

	retryQ, err := channel.QueueDeclare(
		retryQueue, true, false, false, false, retryArgs,
	)
	if err != nil {
		return fmt.Errorf("declare retry queue: %w", err)
	}

	if err := channel.QueueBind(
		retryQ.Name, queue, retryExchange, false, nil,
	); err != nil {
		return fmt.Errorf("bind retry queue: %w", err)
	}

	mainArgs := amqp.Table{
		"x-dead-letter-exchange": dlxExchange,
	}

	if _, err := channel.QueueDeclare(
		queue, true, false, false, false, mainArgs,
	); err != nil {
		return fmt.Errorf("declare main queue: %w", err)
	}

	return nil
}
