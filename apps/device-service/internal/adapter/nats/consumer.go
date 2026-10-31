package natsadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"deviceservice/internal/domain"
	"deviceservice/internal/usecase"

	"github.com/nats-io/nats.go"
)

const (
	deviceRegisteredSubject = "device.registered"
	warmhouseStream         = "WARMHOUSE"
	deviceConsumerName      = "device-service"
)

type Consumer struct {
	url     string
	service *usecase.Service
	logger  *slog.Logger
	conn    *nats.Conn
	sub     *nats.Subscription
}

type DeviceRegisteredEvent struct {
	EventID    string         `json:"eventId"`
	ExternalID string         `json:"external_id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Location   string         `json:"location"`
	Unit       string         `json:"unit"`
	Status     string         `json:"status"`
	State      map[string]any `json:"state"`
}

func NewConsumer(url string, service *usecase.Service, logger *slog.Logger) *Consumer {
	return &Consumer{url: url, service: service, logger: logger}
}

func (c *Consumer) Start(ctx context.Context) error {
	if c.url == "" {
		return nil
	}

	conn, err := nats.Connect(
		c.url,
		nats.Name("warmhouse-device-service"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return fmt.Errorf("create NATS JetStream context: %w", err)
	}
	streamConfig := &nats.StreamConfig{
		Name:     warmhouseStream,
		Subjects: []string{"telemetry.recorded", deviceRegisteredSubject},
	}
	if _, err := js.AddStream(streamConfig); err != nil {
		if _, updateErr := js.UpdateStream(streamConfig); updateErr != nil {
			conn.Close()
			return fmt.Errorf("create NATS stream: %w", err)
		}
	}

	sub, err := js.QueueSubscribeSync(
		deviceRegisteredSubject,
		deviceConsumerName,
		nats.Durable(deviceConsumerName),
		nats.ManualAck(),
	)
	if err != nil {
		conn.Close()
		return fmt.Errorf("subscribe to %s: %w", deviceRegisteredSubject, err)
	}

	c.conn = conn
	c.sub = sub
	go c.consume(ctx)
	return nil
}

func (c *Consumer) consume(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		message, err := c.sub.NextMsg(1 * time.Second)
		if err != nil {
			if err == nats.ErrTimeout {
				continue
			}
			c.logger.Error("failed to receive device event", "error", err)
			continue
		}

		var event DeviceRegisteredEvent
		if err := json.Unmarshal(message.Data, &event); err != nil {
			c.logger.Error("failed to decode device event", "error", err)
			_ = message.Term()
			continue
		}

		_, err = c.service.Register(ctx, domain.DeviceInput{
			ExternalID: event.ExternalID,
			Name:       event.Name,
			Type:       event.Type,
			Location:   event.Location,
			Unit:       event.Unit,
			Status:     event.Status,
			State:      event.State,
		})
		if err != nil {
			c.logger.Error("failed to process device event", "error", err)
			_ = message.Nak()
			continue
		}

		_ = message.Ack()
	}
}

func (c *Consumer) Close() {
	if c.sub != nil {
		_ = c.sub.Unsubscribe()
	}
	if c.conn != nil {
		_ = c.conn.Drain()
	}
}
