package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/johnfarrell/runeplan/application/catalog"
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	"github.com/johnfarrell/runeplan/domain/goal"
	"github.com/johnfarrell/runeplan/domain/skill"
)

// CatalogRepository implements catalog.Repository using PostgreSQL.
type CatalogRepository struct {
	pool *pgxpool.Pool
}

func NewCatalogRepository(pool *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{pool: pool}
}

func (r *CatalogRepository) ListAll(ctx context.Context) ([]domcatalog.Goal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, canonical_key, title, type, COALESCE(description, ''), created_at
		FROM catalog_goals ORDER BY type, title`)
	if err != nil {
		return nil, fmt.Errorf("catalog: list all: %w", err)
	}
	defer rows.Close()
	return scanGoals(rows)
}

func (r *CatalogRepository) ListByType(ctx context.Context, t goal.Type) ([]domcatalog.Goal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, canonical_key, title, type, COALESCE(description, ''), created_at
		FROM catalog_goals WHERE type = $1 ORDER BY title`, string(t))
	if err != nil {
		return nil, fmt.Errorf("catalog: list by type: %w", err)
	}
	defer rows.Close()
	return scanGoals(rows)
}

func (r *CatalogRepository) GetByID(ctx context.Context, id string) (*domcatalog.Goal, error) {
	var g domcatalog.Goal
	err := r.pool.QueryRow(ctx, `
		SELECT id, canonical_key, title, type, COALESCE(description, ''), created_at
		FROM catalog_goals WHERE id = $1`, id).
		Scan(&g.ID, &g.CanonicalKey, &g.Title, &g.Type, &g.Description, &g.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, catalog.ErrNotFound
		}
		return nil, fmt.Errorf("catalog: get by id: %w", err)
	}

	// Load skill requirements
	skillRows, err := r.pool.Query(ctx,
		`SELECT skill, level FROM catalog_skill_requirements WHERE catalog_goal_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("catalog: skill reqs: %w", err)
	}
	defer skillRows.Close()
	for skillRows.Next() {
		var s string
		var lvl int
		if err := skillRows.Scan(&s, &lvl); err != nil {
			return nil, err
		}
		level, _ := skill.NewLevel(lvl)
		g.SkillReqs = append(g.SkillReqs, domcatalog.SkillRequirement{
			CatalogGoalID: id,
			Skill:         skill.Skill(s),
			Level:         level,
		})
	}

	// Load freeform requirements
	reqRows, err := r.pool.Query(ctx,
		`SELECT id, description FROM catalog_requirements WHERE catalog_goal_id = $1 ORDER BY created_at`, id)
	if err != nil {
		return nil, fmt.Errorf("catalog: requirements: %w", err)
	}
	defer reqRows.Close()
	for reqRows.Next() {
		var req domcatalog.Requirement
		req.CatalogGoalID = id
		if err := reqRows.Scan(&req.ID, &req.Description); err != nil {
			return nil, err
		}
		g.Requirements = append(g.Requirements, req)
	}

	return &g, nil
}

func scanGoals(rows pgx.Rows) ([]domcatalog.Goal, error) {
	var goals []domcatalog.Goal
	for rows.Next() {
		var g domcatalog.Goal
		if err := rows.Scan(&g.ID, &g.CanonicalKey, &g.Title, &g.Type, &g.Description, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}
