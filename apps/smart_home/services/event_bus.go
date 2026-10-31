package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

const (
	telemetryRecordedSubject = "telemetry.recorded"
	deviceRegisteredSubject  = "device.registered"
	warmhouseStream          = "WARMHOUSE"
)

type EventBus struct {
	url        string
	connection *nats.Conn
	js         nats.JetStreamContext
	mu         sync.Mutex
}

type EventHeaders struct {
	EventID    string `json:"eventId"`
	EventType  string `json:"eventType"`
	OccurredAt string `json:"occurredAt"`
	TraceID    string `json:"traceId"`
}

type DeviceRegisteredEvent struct {
	EventHeaders
	ExternalID string         `json:"external_id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Location   string         `json:"location"`
	Unit       string         `json:"unit"`
	Status     string         `json:"status"`
	State      map[string]any `json:"state"`
}

type TelemetryRecordedEvent struct {
	EventHeaders
	DeviceID   string    `json:"deviceId"`
	Metric     string    `json:"metric"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	RecordedAt time.Time `json:"recordedAt"`
}

func NewEventBus(url string) *EventBus {
	return &EventBus{url: url}
}

func (b *EventBus) Connect(ctx context.Context) error {
	if b.url == "" {
		return nil
	}

	connection, err := nats.Connect(
		b.url,
		nats.Name("warmhouse-monolith"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.Timeout(2*time.Second),
	)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := connection.JetStream()
	if err != nil {
		connection.Close()
		return fmt.Errorf("create NATS JetStream context: %w", err)
	}

	streamConfig := &nats.StreamConfig{
		Name:     warmhouseStream,
		Subjects: []string{telemetryRecordedSubject, deviceRegisteredSubject},
	}
	if _, err := js.AddStream(streamConfig); err != nil {
		if _, updateErr := js.UpdateStream(streamConfig); updateErr != nil {
			connection.Close()
			return fmt.Errorf("create NATS stream: %w", err)
		}
	}

	b.mu.Lock()
	b.connection = connection
	b.js = js
	b.mu.Unlock()

	return nil
}

func (b *EventBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.connection != nil {
		_ = b.connection.Drain()
	}
}

func (b *EventBus) PublishDeviceRegistered(ctx context.Context, payload DeviceRegisteredEvent) error {
	payload.EventHeaders = newEventHeaders(deviceRegisteredSubject)
	return b.publish(ctx, deviceRegisteredSubject, payload)
}

func (b *EventBus) PublishTelemetry(ctx context.Context, payload TelemetryRecordedEvent) error {
	payload.EventHeaders = newEventHeaders(telemetryRecordedSubject)
	return b.publish(ctx, telemetryRecordedSubject, payload)
}

func (b *EventBus) publish(ctx context.Context, subject string, payload any) error {
	b.mu.Lock()
	js := b.js
	b.mu.Unlock()
	if js == nil {
		return fmt.Errorf("NATS event bus is not connected")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode event: %w", err)
	}

	if _, err := js.PublishMsg(&nats.Msg{Subject: subject, Data: data}, nats.Context(ctx)); err != nil {
		return fmt.Errorf("publish %s: %w", subject, err)
	}
	return nil
}

func newEventHeaders(eventType string) EventHeaders {
	traceID := os.Getenv("TRACE_ID")
	if traceID == "" {
		traceID = uuid.NewString()
	}
	return EventHeaders{
		EventID:    uuid.NewString(),
		EventType:  eventType,
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		TraceID:    traceID,
	}
}
