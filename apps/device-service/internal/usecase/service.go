package usecase

import (
	"context"
	"time"

	"deviceservice/internal/domain"
)

type Repository interface {
	Upsert(ctx context.Context, input domain.DeviceInput) (domain.Device, error)
	List(ctx context.Context) ([]domain.Device, error)
	GetByExternalID(ctx context.Context, externalID string) (domain.Device, error)
	UpdateState(ctx context.Context, externalID string, status string, state map[string]any) (domain.Device, error)
	Delete(ctx context.Context, externalID string) error
	CreateCommand(ctx context.Context, deviceID int64, input domain.CommandInput) (domain.Command, error)
	ListCommands(ctx context.Context, deviceID int64) ([]domain.Command, error)
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

func (s *Service) Register(ctx context.Context, input domain.DeviceInput) (domain.Device, error) {
	input.Normalize()
	if err := input.Validate(); err != nil {
		return domain.Device{}, err
	}
	return s.repository.Upsert(ctx, input)
}

func (s *Service) List(ctx context.Context) ([]domain.Device, error) {
	return s.repository.List(ctx)
}

func (s *Service) Get(ctx context.Context, externalID string) (domain.Device, error) {
	return s.repository.GetByExternalID(ctx, externalID)
}

func (s *Service) UpdateState(ctx context.Context, externalID string, input domain.StateInput) (domain.Device, error) {
	input.Normalize()
	return s.repository.UpdateState(ctx, externalID, input.Status, input.State)
}

func (s *Service) Delete(ctx context.Context, externalID string) error {
	return s.repository.Delete(ctx, externalID)
}

func (s *Service) CreateCommand(ctx context.Context, externalID string, input domain.CommandInput) (domain.Command, error) {
	input.Normalize()
	if err := input.Validate(); err != nil {
		return domain.Command{}, err
	}

	device, err := s.repository.GetByExternalID(ctx, externalID)
	if err != nil {
		return domain.Command{}, err
	}

	return s.repository.CreateCommand(ctx, device.ID, input)
}

func (s *Service) ListCommands(ctx context.Context, externalID string) ([]domain.Command, error) {
	device, err := s.repository.GetByExternalID(ctx, externalID)
	if err != nil {
		return nil, err
	}
	return s.repository.ListCommands(ctx, device.ID)
}
