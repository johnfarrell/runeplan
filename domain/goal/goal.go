package goal

import (
	"time"

	"github.com/johnfarrell/runeplan/domain/skill"
)

// Type classifies what kind of achievement a Goal represents.
type Type string

const (
	TypeQuest  Type = "quest"
	TypeDiary  Type = "diary"
	TypeSkill  Type = "skill"
	TypeBossKC Type = "boss_kc"
	TypeItem   Type = "item"
	TypeCustom Type = "custom"
)

func (t Type) Valid() bool {
	switch t {
	case TypeQuest, TypeDiary, TypeSkill, TypeBossKC, TypeItem, TypeCustom:
		return true
	}
	return false
}

// SkillThreshold is the minimum level required in a Skill to satisfy a goal.
type SkillThreshold struct {
	Skill skill.Skill
	XP    skill.XP
	Level skill.Level
}

func NewSkillLevelThreshold(s skill.Skill, level int) (SkillThreshold, error) {
	if !s.Valid() {
		return SkillThreshold{}, skill.ErrInvalidSkill
	}
	l, err := skill.NewLevel(level)
	if err != nil {
		return SkillThreshold{}, err
	}
	minXP, _ := skill.XPRangeForLevel(level)
	xp, err := skill.NewXP(minXP)
	if err != nil {
		return SkillThreshold{}, err
	}
	return SkillThreshold{Skill: s, XP: xp, Level: l}, nil
}

func (s SkillThreshold) IsSatisfiedByLevel(current skill.Level) bool {
	return current.Value() >= s.Level.Value()
}

func (s SkillThreshold) IsSatisfiedByXP(current skill.XP) bool {
	return current.Value() >= s.XP.Value()
}

// RequirementProgress tracks a user's completion state for a catalog requirement.
type RequirementProgress struct {
	GoalID        string
	RequirementID string
	Description   string // denormalised for display
	Completed     bool
	CompletedAt   *time.Time
}

// CustomRequirement is a user-added freeform requirement on a goal.
type CustomRequirement struct {
	ID          string
	GoalID      string
	Description string
	Completed   bool
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Goal is a per-RSN activated goal. It references a catalog goal and tracks progress.
type Goal struct {
	ID          string
	RSNID       string
	CatalogID   *string
	Title       string
	Type        Type
	Notes       string
	Completed   bool
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Loaded on demand
	Requirements       []RequirementProgress
	CustomRequirements []CustomRequirement
}

func (g *Goal) Complete(at time.Time) {
	g.Completed = true
	g.CompletedAt = &at
	g.UpdatedAt = at
}
