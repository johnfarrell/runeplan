package catalog

import (
	"time"

	"github.com/johnfarrell/runeplan/domain/goal"
	"github.com/johnfarrell/runeplan/domain/skill"
)

// Goal is a pre-seeded canonical OSRS goal (quest, diary, etc.).
type Goal struct {
	ID           string
	CanonicalKey string
	Title        string
	Type         goal.Type
	Description  string
	CreatedAt    time.Time

	Requirements    []Requirement
	SkillReqs       []SkillRequirement
	ItemReqs        []ItemRequirement
	BossReqs        []BossRequirement
	PrerequisiteIDs []string
}

// Requirement is a freeform checklist item on a catalog goal.
type Requirement struct {
	ID            string
	CatalogGoalID string
	Description   string
	CreatedAt     time.Time
}

// SkillRequirement is a minimum skill level required by a catalog goal.
type SkillRequirement struct {
	CatalogGoalID string
	Skill         skill.Skill
	Level         skill.Level
}

// ItemRequirement is an item quantity required by a catalog goal.
type ItemRequirement struct {
	ID            string
	CatalogGoalID string
	ItemName      string
	Quantity      int
}

// BossRequirement is a minimum KC required by a catalog goal.
type BossRequirement struct {
	ID            string
	CatalogGoalID string
	BossName      string
	KC            int
}
