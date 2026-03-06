package skill_test

import (
	"testing"

	domskill "github.com/johnfarrell/runeplan/domain/skill"
	"github.com/johnfarrell/runeplan/application/skill"
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
)

func TestAggregateThresholds_MaxPerSkill(t *testing.T) {
	goals := []domgoal.Goal{
		{RSNID: "r1", Completed: false},
		{RSNID: "r1", Completed: false},
	}
	catalogGoals := []domcatalog.Goal{
		{SkillReqs: []domcatalog.SkillRequirement{
			{Skill: domskill.Agility, Level: mustLevel(60)},
		}},
		{SkillReqs: []domcatalog.SkillRequirement{
			{Skill: domskill.Agility, Level: mustLevel(70)},
			{Skill: domskill.Magic, Level: mustLevel(55)},
		}},
	}
	current := map[domskill.Skill]domskill.XP{
		domskill.Agility: mustXP(302288), // level 61
	}

	thresholds := skill.AggregateThresholds(goals, catalogGoals, current)

	agility, ok := thresholds[domskill.Agility]
	if !ok {
		t.Fatal("expected agility threshold")
	}
	if agility.Required.Value() != 70 {
		t.Errorf("agility required: got %d, want 70", agility.Required.Value())
	}
	if agility.Satisfied {
		t.Error("agility should not be satisfied (current 61 < required 70)")
	}
	magic, ok := thresholds[domskill.Magic]
	if !ok {
		t.Fatal("expected magic threshold")
	}
	if magic.Required.Value() != 55 {
		t.Errorf("magic required: got %d, want 55", magic.Required.Value())
	}
}

func mustLevel(v int) domskill.Level {
	l, err := domskill.NewLevel(v)
	if err != nil {
		panic(err)
	}
	return l
}

func mustXP(v int) domskill.XP {
	x, err := domskill.NewXP(v)
	if err != nil {
		panic(err)
	}
	return x
}
