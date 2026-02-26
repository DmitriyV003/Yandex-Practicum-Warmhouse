package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName = "warmhouse"
	QueueName    = "device.telemetry.events"
)

type Event struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

type Broker struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func New(url string) (*Broker, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}

	if err := ch.ExchangeDeclare(ExchangeName, "topic", true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	q, err := ch.QueueDeclare(QueueName, true, false, false, false, nil)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	for _, key := range []string{"telemetry.updated", "telemetry.no_data"} {
		if err := ch.QueueBind(q.Name, key, ExchangeName, false, nil); err != nil {
			ch.Close()
			conn.Close()
			return nil, fmt.Errorf("bind queue: %w", err)
		}
	}

	return &Broker{conn: conn, channel: ch}, nil
}

func NewWithRetry(url string, maxRetries int) (*Broker, error) {
	var b *Broker
	var err error
	for i := 0; i < maxRetries; i++ {
		b, err = New(url)
		if err == nil {
			return b, nil
		}
		log.Printf("rabbitmq not ready, retry %d/%d: %v", i+1, maxRetries, err)
		time.Sleep(time.Duration(i+1) * 2 * time.Second)
	}
	return nil, fmt.Errorf("rabbitmq connect failed after %d retries: %w", maxRetries, err)
}

func (b *Broker) Publish(ctx context.Context, routingKey string, payload map[string]interface{}) error {
	event := Event{
		Type:      routingKey,
		Timestamp: time.Now(),
		Payload:   payload,
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return b.channel.PublishWithContext(ctx, ExchangeName, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (b *Broker) Consume(handler func(Event)) error {
	msgs, err := b.channel.Consume(QueueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	go func() {
		for d := range msgs {
			var event Event
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Printf("unmarshal event error: %v", err)
				d.Nack(false, true)
				continue
			}
			handler(event)
			d.Ack(false)
		}
	}()

	return nil
}

func (b *Broker) Close() {
	if b.channel != nil {
		b.channel.Close()
	}
	if b.conn != nil {
		b.conn.Close()
	}
}
