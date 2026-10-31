package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"deviceservice/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const deviceColumns = `
	id, external_id, name, type, location, unit, status,
	desired_state, reported_state, created_at, updated_at
`

type Repository struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Repository, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Repository{pool: pool}, nil
}

func (r *Repository) Close() {
	r.pool.Close()
}

func (r *Repository) Upsert(ctx context.Context, input domain.DeviceInput) (domain.Device, error) {
	state, err := encodeJSON(input.State)
	if err != nil {
		return domain.Device{}, err
	}

	query := `
		INSERT INTO devices (
			external_id, name, type, location, unit, status, desired_state, reported_state
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		ON CONFLICT (external_id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			location = EXCLUDED.location,
			unit = EXCLUDED.unit,
			status = CASE
				WHEN EXCLUDED.status <> 'offline' THEN EXCLUDED.status
				ELSE devices.status
			END,
			desired_state = CASE
				WHEN EXCLUDED.desired_state = '{}'::jsonb THEN devices.desired_state
				ELSE EXCLUDED.desired_state
			END,
			reported_state = CASE
				WHEN EXCLUDED.reported_state = '{}'::jsonb THEN devices.reported_state
				ELSE EXCLUDED.reported_state
			END,
			updated_at = NOW()
		RETURNING ` + deviceColumns

	device, err := scanDevice(r.pool.QueryRow(ctx, query,
		input.ExternalID,
		input.Name,
		input.Type,
		input.Location,
		input.Unit,
		input.Status,
		state,
	))
	if err != nil {
		return domain.Device{}, fmt.Errorf("upsert device: %w", err)
	}

	return device, nil
}

func (r *Repository) List(ctx context.Context) ([]domain.Device, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+deviceColumns+` FROM devices ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query devices: %w", err)
	}
	defer rows.Close()

	devices := make([]domain.Device, 0)
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		devices = append(devices, device)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate devices: %w", err)
	}

	return devices, nil
}

func (r *Repository) GetByExternalID(ctx context.Context, externalID string) (domain.Device, error) {
	query := `SELECT ` + deviceColumns + ` FROM devices WHERE external_id = $1`

	device, err := scanDevice(r.pool.QueryRow(ctx, query, externalID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Device{}, domain.ErrDeviceNotFound
	}
	if err != nil {
		return domain.Device{}, fmt.Errorf("get device: %w", err)
	}

	return device, nil
}

func (r *Repository) UpdateState(ctx context.Context, externalID string, status string, state map[string]any) (domain.Device, error) {
	payload, err := encodeJSON(state)
	if err != nil {
		return domain.Device{}, err
	}

	query := `
		UPDATE devices
		SET status = COALESCE(NULLIF($2, ''), status),
			reported_state = reported_state || $3::jsonb,
			updated_at = NOW()
		WHERE external_id = $1
		RETURNING ` + deviceColumns

	device, err := scanDevice(r.pool.QueryRow(ctx, query, externalID, status, payload))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Device{}, domain.ErrDeviceNotFound
	}
	if err != nil {
		return domain.Device{}, fmt.Errorf("update device state: %w", err)
	}

	return device, nil
}

func (r *Repository) Delete(ctx context.Context, externalID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM devices WHERE external_id = $1`, externalID)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDeviceNotFound
	}
	return nil
}

func (r *Repository) CreateCommand(ctx context.Context, deviceID int64, input domain.CommandInput) (domain.Command, error) {
	payload, err := encodeJSON(input.Payload)
	if err != nil {
		return domain.Command{}, err
	}

	query := `
		INSERT INTO commands (device_id, type, payload)
		VALUES ($1, $2, $3)
		RETURNING id, device_id, type, payload, status, created_at
	`

	var command domain.Command
	var rawPayload []byte
	err = r.pool.QueryRow(ctx, query, deviceID, input.Type, payload).Scan(
		&command.ID,
		&command.DeviceID,
		&command.Type,
		&rawPayload,
		&command.Status,
		&command.CreatedAt,
	)
	if err != nil {
		return domain.Command{}, fmt.Errorf("create command: %w", err)
	}

	command.Payload, err = decodeJSON(rawPayload)
	if err != nil {
		return domain.Command{}, err
	}

	return command, nil
}

func (r *Repository) ListCommands(ctx context.Context, deviceID int64) ([]domain.Command, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, device_id, type, payload, status, created_at
		FROM commands
		WHERE device_id = $1
		ORDER BY id DESC
	`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("query commands: %w", err)
	}
	defer rows.Close()

	commands := make([]domain.Command, 0)
	for rows.Next() {
		var command domain.Command
		var rawPayload []byte
		if err := rows.Scan(
			&command.ID,
			&command.DeviceID,
			&command.Type,
			&rawPayload,
			&command.Status,
			&command.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan command: %w", err)
		}

		command.Payload, err = decodeJSON(rawPayload)
		if err != nil {
			return nil, err
		}
		commands = append(commands, command)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate commands: %w", err)
	}

	return commands, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanDevice(row scanner) (domain.Device, error) {
	var device domain.Device
	var desiredState []byte
	var reportedState []byte

	err := row.Scan(
		&device.ID,
		&device.ExternalID,
		&device.Name,
		&device.Type,
		&device.Location,
		&device.Unit,
		&device.Status,
		&desiredState,
		&reportedState,
		&device.CreatedAt,
		&device.UpdatedAt,
	)
	if err != nil {
		return domain.Device{}, err
	}

	device.DesiredState, err = decodeJSON(desiredState)
	if err != nil {
		return domain.Device{}, err
	}
	device.ReportedState, err = decodeJSON(reportedState)
	if err != nil {
		return domain.Device{}, err
	}

	return device, nil
}

func encodeJSON(value map[string]any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode state: %w", err)
	}
	return encoded, nil
}

func decodeJSON(value []byte) (map[string]any, error) {
	if len(value) == 0 {
		return map[string]any{}, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, nil
}
