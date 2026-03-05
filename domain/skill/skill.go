package skill

import "errors"

var ErrInvalidXP = errors.New("xp cannot be negative")
var ErrInvalidLevel = errors.New("level must be between 1 and 99")
var ErrInvalidSkill = errors.New("invalid skill name")

type Skill string

const (
	Attack       Skill = "attack"
	Strength     Skill = "strength"
	Defence      Skill = "defence"
	Ranged       Skill = "ranged"
	Prayer       Skill = "prayer"
	Magic        Skill = "magic"
	Runecraft    Skill = "runecraft"
	Hitpoints    Skill = "hitpoints"
	Crafting     Skill = "crafting"
	Mining       Skill = "mining"
	Smithing     Skill = "smithing"
	Fishing      Skill = "fishing"
	Cooking      Skill = "cooking"
	Firemaking   Skill = "firemaking"
	Woodcut      Skill = "woodcutting"
	Agility      Skill = "agility"
	Herblore     Skill = "herblore"
	Thieving     Skill = "thieving"
	Fletching    Skill = "fletching"
	Slayer       Skill = "slayer"
	Farming      Skill = "farming"
	Construction Skill = "construction"
	Hunter       Skill = "hunter"
	Sailing      Skill = "sailing"
)

// All is the canonical set of valid OSRS skills.
// Used for validation and iteration (e.g. rendering a skill grid).
var All = []Skill{
	Attack, Strength, Defence, Ranged, Prayer, Magic,
	Runecraft, Hitpoints, Crafting, Mining, Smithing,
	Fishing, Cooking, Firemaking, Woodcut, Agility,
	Herblore, Thieving, Fletching, Slayer, Farming,
	Construction, Hunter, Sailing,
}

var allSkills = func() map[Skill]struct{} {
	m := make(map[Skill]struct{}, len(All))
	for _, s := range All {
		m[s] = struct{}{}
	}
	return m
}()

// Valid returns true if this is a recognised OSRS skill name.
func (s Skill) Valid() bool {
	_, ok := allSkills[s]
	return ok
}

// String satisfies fmt.Stringer.
func (s Skill) String() string { return string(s) }

// XP is a value object representing raw experience points.
// It is always non-negative.
type XP struct{ value int }

func NewXP(v int) (XP, error) {
	if v < 0 {
		return XP{}, ErrInvalidXP
	}
	return XP{value: v}, nil
}

func (x XP) Value() int { return x.value }

// Level is a value object representing a skill level between 1 and 99.
type Level struct{ value int }

func NewLevel(v int) (Level, error) {
	if v < 1 || v > 126 {
		return Level{}, ErrInvalidLevel
	}
	return Level{value: v}, nil
}

func (l Level) Value() int { return l.value }

// ToLevel calculates the OSRS skill level for a given XP value.
// Uses the official OSRS formula — level is determined by the highest
// level whose XP threshold is at or below the given XP.
func (x XP) ToLevel() Level {
	for lvl := 126; lvl >= 1; lvl-- {
		if x.value >= xpTable[lvl] {
			l, _ := NewLevel(lvl)
			return l
		}
	}
	l, _ := NewLevel(1)
	return l
}

// ToXP returns the minimum XP required to reach this level.
func (l Level) ToXP() XP {
	if l.value <= 1 {
		return XP{value: 0}
	}
	return XP{value: xpTable[l.value]}
}

// XPRemaining returns how much XP is needed to reach the target level
// from the current XP. Returns 0 if already at or past the target.
func (x XP) XPRemaining(target Level) int {
	needed := target.ToXP().value
	if x.value >= needed {
		return 0
	}
	return needed - x.value
}
