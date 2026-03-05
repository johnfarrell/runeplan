package user

import (
	"time"

	"github.com/johnfarrell/runeplan/domain/skill"
)

// User is a RunePlan account. It may have zero or more linked OSRS accounts (RSNs).
type User struct {
	ID        string
	RSNs      []RSN
	CreatedAt time.Time
}

// ActiveRSN returns the first RSN if any exist.
func (u *User) ActiveRSN() *RSN {
	if len(u.RSNs) == 0 {
		return nil
	}
	return &u.RSNs[0]
}

// RSN is a linked OSRS account. Skill levels are keyed by skill name.
type RSN struct {
	ID          string
	UserID      string
	RSN         string
	SkillLevels map[skill.Skill]skill.XP
	SyncedAt    *time.Time
	CreatedAt   time.Time
}
