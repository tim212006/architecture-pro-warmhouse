package usecase

import (
	"context"
	"time"

	"telemetryservice/internal/domain"
)

type Repository interface {
	Insert(ctx context.Context, reading domain.Reading) (domain.Reading, error)
	List(ctx context.Context, filter domain.Filter) ([]domain.Reading, error)
	Latest(ctx context.Context, deviceID string, metric string) (domain.Reading, error)
}

type Clock interface {
	Now() time.Time
}

type Service struct {
	repository Repository
	clock      Clock
}

func NewService(repository Repository, clock Clock) *Service {
	return &Service{
		repository: repository,
		clock:      clock,
	}
}

func (s *Service) Record(ctx context.Context, input domain.RecordInput) (domain.Reading, error) {
	input.Normalize()
	if err := input.Validate(); err != nil {
		return domain.Reading{}, err
	}

	recordedAt := s.clock.Now()
	if input.RecordedAt != nil {
		recordedAt = *input.RecordedAt
	}

	return s.repository.Insert(ctx, domain.Reading{
		DeviceID:   input.DeviceID,
		Metric:     input.Metric,
		Value:      input.Value,
		Unit:       input.Unit,
		RecordedAt: recordedAt,
	})
}

func (s *Service) List(ctx context.Context, filter domain.Filter) ([]domain.Reading, error) {
	return s.repository.List(ctx, filter.Normalize())
}

func (s *Service) Latest(ctx context.Context, deviceID string, metric string) (domain.Reading, error) {
	return s.repository.Latest(ctx, deviceID, metric)
}
