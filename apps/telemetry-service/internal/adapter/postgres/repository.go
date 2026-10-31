package postgres

import (
	"context"
	"errors"
	"fmt"

	"telemetryservice/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const readingColumns = `
	id, device_id, metric, value, unit, recorded_at, created_at
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

func (r *Repository) Insert(ctx context.Context, reading domain.Reading) (domain.Reading, error) {
	query := `
		INSERT INTO telemetry_data (device_id, metric, value, unit, recorded_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + readingColumns

	stored, err := scanReading(r.pool.QueryRow(ctx, query,
		reading.DeviceID,
		reading.Metric,
		reading.Value,
		reading.Unit,
		reading.RecordedAt,
	))
	if err != nil {
		return domain.Reading{}, fmt.Errorf("insert telemetry: %w", err)
	}

	return stored, nil
}

func (r *Repository) List(ctx context.Context, filter domain.Filter) ([]domain.Reading, error) {
	query := `SELECT ` + readingColumns + ` FROM telemetry_data WHERE 1 = 1`
	args := make([]any, 0, 5)

	if filter.DeviceID != "" {
		args = append(args, filter.DeviceID)
		query += fmt.Sprintf(" AND device_id = $%d", len(args))
	}
	if filter.Metric != "" {
		args = append(args, filter.Metric)
		query += fmt.Sprintf(" AND metric = $%d", len(args))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		query += fmt.Sprintf(" AND recorded_at >= $%d", len(args))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		query += fmt.Sprintf(" AND recorded_at <= $%d", len(args))
	}

	args = append(args, filter.Limit)
	query += fmt.Sprintf(" ORDER BY recorded_at DESC, id DESC LIMIT $%d", len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query telemetry: %w", err)
	}
	defer rows.Close()

	readings := make([]domain.Reading, 0)
	for rows.Next() {
		reading, err := scanReading(rows)
		if err != nil {
			return nil, fmt.Errorf("scan telemetry: %w", err)
		}
		readings = append(readings, reading)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate telemetry: %w", err)
	}

	return readings, nil
}

func (r *Repository) Latest(ctx context.Context, deviceID string, metric string) (domain.Reading, error) {
	query := `
		SELECT ` + readingColumns + `
		FROM telemetry_data
		WHERE device_id = $1 AND ($2 = '' OR metric = $2)
		ORDER BY recorded_at DESC, id DESC
		LIMIT 1
	`

	reading, err := scanReading(r.pool.QueryRow(ctx, query, deviceID, metric))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Reading{}, domain.ErrReadingNotFound
	}
	if err != nil {
		return domain.Reading{}, fmt.Errorf("get latest telemetry: %w", err)
	}

	return reading, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanReading(row scanner) (domain.Reading, error) {
	var reading domain.Reading
	err := row.Scan(
		&reading.ID,
		&reading.DeviceID,
		&reading.Metric,
		&reading.Value,
		&reading.Unit,
		&reading.RecordedAt,
		&reading.CreatedAt,
	)
	if err != nil {
		return domain.Reading{}, err
	}
	return reading, nil
}
