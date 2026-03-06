package skill

import (
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
	domskill "github.com/johnfarrell/runeplan/domain/skill"
)

// Threshold holds the highest required level for a skill and whether the user satisfies it.
type Threshold struct {
	Skill     domskill.Skill
	Required  domskill.Level
	Current   domskill.Level
	CurrentXP domskill.XP
	XPNeeded  int
	Satisfied bool
}

// AggregateThresholds computes per-skill max required level across all active goals,
// compared against the current XP map.
//
// goals and catalogGoals must be parallel slices (goals[i] corresponds to catalogGoals[i]).
// goals with Completed=true are excluded.
func AggregateThresholds(
	goals []domgoal.Goal,
	catalogGoals []domcatalog.Goal,
	current map[domskill.Skill]domskill.XP,
) map[domskill.Skill]Threshold {
	maxLevel := make(map[domskill.Skill]domskill.Level)

	for i, g := range goals {
		if g.Completed {
			continue
		}
		if i >= len(catalogGoals) {
			continue
		}
		for _, sr := range catalogGoals[i].SkillReqs {
			if existing, ok := maxLevel[sr.Skill]; !ok || sr.Level.Value() > existing.Value() {
				maxLevel[sr.Skill] = sr.Level
			}
		}
	}

	thresholds := make(map[domskill.Skill]Threshold, len(maxLevel))
	for s, required := range maxLevel {
		currentXP, _ := current[s]
		currentLevel := currentXP.ToLevel()
		xpNeeded := currentXP.XPRemaining(required)
		thresholds[s] = Threshold{
			Skill:     s,
			Required:  required,
			Current:   currentLevel,
			CurrentXP: currentXP,
			XPNeeded:  xpNeeded,
			Satisfied: xpNeeded == 0,
		}
	}
	return thresholds
}
