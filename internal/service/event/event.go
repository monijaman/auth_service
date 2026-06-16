package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type EventType string

const (
	UserRegistered  EventType = "UserRegistered"
	UserLoggedIn    EventType = "UserLoggedIn"
	UserLoggedOut   EventType = "UserLoggedOut"
	PasswordChanged EventType = "PasswordChanged"
	EmailVerified   EventType = "EmailVerified"
	UserDeleted     EventType = "UserDeleted"
)

type Event struct {
	Type      EventType   `json:"type"`
	Payload   interface{} `json:"payload"`
	OccuredAt time.Time   `json:"occurred_at"`
}

type Publisher interface {
	Publish(ctx context.Context, e Event) error
	Close() error
}

type RabbitPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	exchange string
}

func NewRabbitPublisher(url, exchange string) (*RabbitPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("rabbitmq exchange declare: %w", err)
	}
	return &RabbitPublisher{conn: conn, channel: ch, exchange: exchange}, nil
}

func (p *RabbitPublisher) Publish(_ context.Context, e Event) error {
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return p.channel.Publish(
		p.exchange,
		string(e.Type),
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
}

func (p *RabbitPublisher) Close() error {
	if err := p.channel.Close(); err != nil {
		return err
	}
	return p.conn.Close()
}

// NoopPublisher silently drops events — use when RabbitMQ is disabled.
type NoopPublisher struct{}

func (NoopPublisher) Publish(_ context.Context, _ Event) error { return nil }
func (NoopPublisher) Close() error                              { return nil }
