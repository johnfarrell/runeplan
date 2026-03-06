package catalog

import (
	"context"
	"errors"

	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	"github.com/johnfarrell/runeplan/domain/goal"
)

var ErrNotFound = errors.New("catalog: goal not found")

// Repository is the persistence interface for catalog goals.
type Repository interface {
	ListAll(ctx context.Context) ([]domcatalog.Goal, error)
	ListByType(ctx context.Context, t goal.Type) ([]domcatalog.Goal, error)
	GetByID(ctx context.Context, id string) (*domcatalog.Goal, error)
}

// Service handles catalog browsing use cases.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListAll(ctx context.Context) ([]domcatalog.Goal, error) {
	return s.repo.ListAll(ctx)
}

func (s *Service) ListByType(ctx context.Context, t goal.Type) ([]domcatalog.Goal, error) {
	return s.repo.ListByType(ctx, t)
}

func (s *Service) GetByID(ctx context.Context, id string) (*domcatalog.Goal, error) {
	return s.repo.GetByID(ctx, id)
}
