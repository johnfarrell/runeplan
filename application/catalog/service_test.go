package catalog_test

import (
	"context"
	"testing"

	"github.com/johnfarrell/runeplan/application/catalog"
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	"github.com/johnfarrell/runeplan/domain/goal"
)

type mockRepo struct {
	goals []domcatalog.Goal
}

func (m *mockRepo) ListByType(ctx context.Context, t goal.Type) ([]domcatalog.Goal, error) {
	var out []domcatalog.Goal
	for _, g := range m.goals {
		if g.Type == t {
			out = append(out, g)
		}
	}
	return out, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*domcatalog.Goal, error) {
	for _, g := range m.goals {
		if g.ID == id {
			return &g, nil
		}
	}
	return nil, catalog.ErrNotFound
}

func (m *mockRepo) ListAll(ctx context.Context) ([]domcatalog.Goal, error) {
	return m.goals, nil
}

func TestListByType(t *testing.T) {
	repo := &mockRepo{goals: []domcatalog.Goal{
		{ID: "1", Type: goal.TypeQuest, Title: "Q"},
		{ID: "2", Type: goal.TypeDiary, Title: "D"},
	}}
	svc := catalog.NewService(repo)
	got, err := svc.ListByType(context.Background(), goal.TypeQuest)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "1" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &mockRepo{}
	svc := catalog.NewService(repo)
	_, err := svc.GetByID(context.Background(), "missing")
	if err != catalog.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
