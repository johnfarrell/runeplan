package goal

import (
	"time"

	"github.com/johnfarrell/runeplan/domain/skill"
)

// Type classifies what kind of achievement a Goal represents.
// This drives how completion is determined and how the UI presents it.
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

// SkillThreshold is the minimum level or XP required in a Skill to satisfy
// a goal's Skill requirement.
// Thresholds are global per-user per-skill: achieving level 72 agility
// satisfies every threshold at or below 72 across all active goals.
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
	return s.Level.Value() > current.Value()
}

func (s SkillThreshold) IsSatisfiedByXP(current skill.XP) bool {
	return s.XP.Value() > current.Value()
}

type Requirement struct {
	ID          string
	GoalID      string
	Description string
	Completed   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

func (r *Requirement) Complete(at time.Time) {
	r.Completed = true
	r.UpdatedAt = at
	r.CompletedAt = &at
}

func (r *Requirement) Reopen(at time.Time) {
	r.Completed = false
	r.CompletedAt = nil
	r.UpdatedAt = at
}

type Goal struct {
	ID          string
	UserID      string
	CatalogID   *string
	Title       string
	Type        Type
	Completed   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time

	Requirements    []Requirement
	SkillThresholds []SkillThreshold
	PrerequisiteIDs []string
}
