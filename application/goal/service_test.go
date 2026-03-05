package goal_test

import (
	"context"
	"testing"
	"time"

	"github.com/johnfarrell/runeplan/application/goal"
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
)

type mockGoalRepo struct {
	goals []domgoal.Goal
}

func (m *mockGoalRepo) ListByRSN(ctx context.Context, rsnID string) ([]domgoal.Goal, error) {
	var out []domgoal.Goal
	for _, g := range m.goals {
		if g.RSNID == rsnID {
			out = append(out, g)
		}
	}
	return out, nil
}
func (m *mockGoalRepo) Activate(ctx context.Context, rsnID, catalogID string) (*domgoal.Goal, error) {
	g := &domgoal.Goal{
		ID:        "new-id",
		RSNID:     rsnID,
		Title:     "Test Goal",
		Type:      domgoal.TypeQuest,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	cid := catalogID
	g.CatalogID = &cid
	m.goals = append(m.goals, *g)
	return g, nil
}
func (m *mockGoalRepo) Complete(ctx context.Context, goalID string) error {
	for i, g := range m.goals {
		if g.ID == goalID {
			m.goals[i].Completed = true
			return nil
		}
	}
	return goal.ErrNotFound
}
func (m *mockGoalRepo) ToggleRequirement(ctx context.Context, goalID, requirementID string) (bool, error) {
	return true, nil
}

func TestActivate(t *testing.T) {
	repo := &mockGoalRepo{}
	svc := goal.NewService(repo)
	g, err := svc.Activate(context.Background(), "rsn1", "catalog1")
	if err != nil {
		t.Fatal(err)
	}
	if g.RSNID != "rsn1" {
		t.Errorf("got RSNID %q, want rsn1", g.RSNID)
	}
}

func TestListByRSN(t *testing.T) {
	repo := &mockGoalRepo{}
	svc := goal.NewService(repo)
	_, _ = svc.Activate(context.Background(), "rsn1", "cat1")
	goals, err := svc.List(context.Background(), "rsn1")
	if err != nil {
		t.Fatal(err)
	}
	if len(goals) != 1 {
		t.Errorf("got %d goals, want 1", len(goals))
	}
}
