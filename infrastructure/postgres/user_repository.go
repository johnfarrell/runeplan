package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	domskill "github.com/johnfarrell/runeplan/domain/skill"
)

// UserRepository implements the user application RSNRepository.
type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// UpdateSkillLevels serialises levels as JSONB and updates user_rsns.skill_levels.
func (r *UserRepository) UpdateSkillLevels(ctx context.Context, rsnID string, levels map[domskill.Skill]domskill.XP) error {
	raw := make(map[string]int, len(levels))
	for s, xp := range levels {
		raw[string(s)] = xp.Value()
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("user: marshal skill levels: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE user_rsns SET skill_levels = $1, synced_at = now() WHERE id = $2`, b, rsnID)
	return err
}
