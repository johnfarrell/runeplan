package goal

import (
	"context"
	"errors"

	domgoal "github.com/johnfarrell/runeplan/domain/goal"
)

var ErrNotFound = errors.New("goal: not found")

// Repository is the persistence interface for user goals.
type Repository interface {
	ListByRSN(ctx context.Context, rsnID string) ([]domgoal.Goal, error)
	Activate(ctx context.Context, rsnID, catalogID string) (*domgoal.Goal, error)
	Complete(ctx context.Context, goalID string) error
	ToggleRequirement(ctx context.Context, goalID, requirementID string) (completed bool, err error)
}

// Service handles goal planning use cases.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, rsnID string) ([]domgoal.Goal, error) {
	return s.repo.ListByRSN(ctx, rsnID)
}

func (s *Service) Activate(ctx context.Context, rsnID, catalogID string) (*domgoal.Goal, error) {
	return s.repo.Activate(ctx, rsnID, catalogID)
}

func (s *Service) Complete(ctx context.Context, goalID string) error {
	return s.repo.Complete(ctx, goalID)
}

func (s *Service) ToggleRequirement(ctx context.Context, goalID, requirementID string) (bool, error) {
	return s.repo.ToggleRequirement(ctx, goalID, requirementID)
}
