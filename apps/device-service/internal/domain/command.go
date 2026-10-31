package domain

import (
	"fmt"
	"strings"
	"time"
)

type Command struct {
	ID        int64          `json:"id"`
	DeviceID  int64          `json:"device_id"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	Status    string         `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
}

type CommandInput struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

func (i *CommandInput) Normalize() {
	i.Type = strings.TrimSpace(i.Type)
	if i.Payload == nil {
		i.Payload = map[string]any{}
	}
}

func (i CommandInput) Validate() error {
	if i.Type == "" {
		return fmt.Errorf("%w: type is required", ErrInvalidInput)
	}
	return nil
}
