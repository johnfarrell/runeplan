package user

import (
	"context"
	"fmt"

	domskill "github.com/johnfarrell/runeplan/domain/skill"
)

// HiscoresClient fetches skill XP for a given RSN string.
type HiscoresClient interface {
	Fetch(rsn string) (map[domskill.Skill]domskill.XP, error)
}

// RSNRepository persists RSN skill data.
type RSNRepository interface {
	UpdateSkillLevels(ctx context.Context, rsnID string, levels map[domskill.Skill]domskill.XP) error
}

// Service handles user-facing use cases (hiscores sync).
type Service struct {
	hiscores HiscoresClient
	repo     RSNRepository
}

func NewService(hiscores HiscoresClient, repo RSNRepository) *Service {
	return &Service{hiscores: hiscores, repo: repo}
}

// SyncHiscores fetches the latest OSRS hiscore data for rsnName and persists it.
func (s *Service) SyncHiscores(ctx context.Context, rsnID, rsnName string) (map[domskill.Skill]domskill.XP, error) {
	levels, err := s.hiscores.Fetch(rsnName)
	if err != nil {
		return nil, fmt.Errorf("sync hiscores: %w", err)
	}
	if err := s.repo.UpdateSkillLevels(ctx, rsnID, levels); err != nil {
		return nil, fmt.Errorf("sync hiscores: persist: %w", err)
	}
	return levels, nil
}
