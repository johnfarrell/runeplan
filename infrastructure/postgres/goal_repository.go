package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goalapp "github.com/johnfarrell/runeplan/application/goal"
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
)

// GoalRepository implements goal.Repository using PostgreSQL.
type GoalRepository struct {
	pool *pgxpool.Pool
}

func NewGoalRepository(pool *pgxpool.Pool) *GoalRepository {
	return &GoalRepository{pool: pool}
}

func (r *GoalRepository) ListByRSN(ctx context.Context, rsnID string) ([]domgoal.Goal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT g.id, g.rsn_id, g.catalog_id, g.title, g.type, COALESCE(g.notes,''),
		       g.completed, g.completed_at, g.created_at, g.updated_at
		FROM goals g WHERE g.rsn_id = $1 ORDER BY g.created_at`, rsnID)
	if err != nil {
		return nil, fmt.Errorf("goals: list: %w", err)
	}
	defer rows.Close()

	var goals []domgoal.Goal
	for rows.Next() {
		var g domgoal.Goal
		if err := rows.Scan(&g.ID, &g.RSNID, &g.CatalogID, &g.Title, &g.Type, &g.Notes,
			&g.Completed, &g.CompletedAt, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range goals {
		if err := r.loadRequirements(ctx, &goals[i]); err != nil {
			return nil, err
		}
	}
	return goals, nil
}

func (r *GoalRepository) loadRequirements(ctx context.Context, g *domgoal.Goal) error {
	rows, err := r.pool.Query(ctx, `
		SELECT p.requirement_id, cr.description, p.completed, p.completed_at
		FROM goal_requirement_progress p
		JOIN catalog_requirements cr ON cr.id = p.requirement_id
		WHERE p.goal_id = $1 ORDER BY cr.created_at`, g.ID)
	if err != nil {
		return fmt.Errorf("goals: load reqs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rp domgoal.RequirementProgress
		rp.GoalID = g.ID
		if err := rows.Scan(&rp.RequirementID, &rp.Description, &rp.Completed, &rp.CompletedAt); err != nil {
			return err
		}
		g.Requirements = append(g.Requirements, rp)
	}
	return rows.Err()
}

func (r *GoalRepository) Activate(ctx context.Context, rsnID, catalogID string) (*domgoal.Goal, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("goals: activate: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var title, goalType string
	if err := tx.QueryRow(ctx,
		`SELECT title, type FROM catalog_goals WHERE id = $1`, catalogID).
		Scan(&title, &goalType); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("goals: activate: catalog goal not found")
		}
		return nil, fmt.Errorf("goals: activate: fetch catalog: %w", err)
	}

	var goalID string
	now := time.Now()
	if err := tx.QueryRow(ctx, `
		INSERT INTO goals (rsn_id, catalog_id, title, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id`, rsnID, catalogID, title, goalType, now).Scan(&goalID); err != nil {
		return nil, fmt.Errorf("goals: activate: insert: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO goal_requirement_progress (goal_id, requirement_id)
		SELECT $1, id FROM catalog_requirements WHERE catalog_goal_id = $2`, goalID, catalogID)
	if err != nil {
		return nil, fmt.Errorf("goals: activate: insert progress: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("goals: activate: commit: %w", err)
	}

	cid := catalogID
	return &domgoal.Goal{
		ID:        goalID,
		RSNID:     rsnID,
		CatalogID: &cid,
		Title:     title,
		Type:      domgoal.Type(goalType),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *GoalRepository) Complete(ctx context.Context, goalID string) error {
	now := time.Now()
	ct, err := r.pool.Exec(ctx,
		`UPDATE goals SET completed = true, completed_at = $1, updated_at = $1 WHERE id = $2`, now, goalID)
	if err != nil {
		return fmt.Errorf("goals: complete: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return goalapp.ErrNotFound
	}
	return nil
}

func (r *GoalRepository) ToggleRequirement(ctx context.Context, goalID, requirementID string) (bool, error) {
	var completed bool
	err := r.pool.QueryRow(ctx, `
		UPDATE goal_requirement_progress
		SET completed = NOT completed,
		    completed_at = CASE WHEN NOT completed THEN now() ELSE NULL END
		WHERE goal_id = $1 AND requirement_id = $2
		RETURNING completed`, goalID, requirementID).Scan(&completed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, goalapp.ErrNotFound
		}
		return false, fmt.Errorf("goals: toggle req: %w", err)
	}
	return completed, nil
}
