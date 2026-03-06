package skill

import "testing"

// ----------------------------------------------------------------------------
// LevelForXP — full virtual level support (1–126)
// ----------------------------------------------------------------------------

func TestLevelForXP_Negative(t *testing.T) {
	// Negative XP is treated as zero — returns level 1.
	if got := LevelForXP(-1); got != 1 {
		t.Errorf("LevelForXP(-1) = %d, want 1", got)
	}
}

func TestLevelForXP_Zero(t *testing.T) {
	if got := LevelForXP(0); got != 1 {
		t.Errorf("LevelForXP(0) = %d, want 1", got)
	}
}

// TestLevelForXP_WikiThresholds verifies every notable level boundary against
// the official OSRS XP table: https://oldschool.runescape.wiki/w/Experience
func TestLevelForXP_WikiThresholds(t *testing.T) {
	cases := []struct {
		xp      int
		wantLvl int
		label   string
	}{
		// Level 1: any XP below the level 2 threshold of 83.
		{0, 1, "level 1 lower bound"},
		{82, 1, "one below level 2"},
		// Level 2 boundary.
		{83, 2, "level 2 threshold"},
		{173, 2, "one below level 3"},
		// Level 10 — sampled milestone from the wiki.
		{1_153, 9, "one below level 10"},
		{1_154, 10, "level 10 threshold"},
		// Level 30.
		{13_362, 29, "one below level 30"},
		{13_363, 30, "level 30 threshold"},
		// Level 50.
		{101_332, 49, "one below level 50"},
		{101_333, 50, "level 50 threshold"},
		// Level 70 — common Achievement Diary skill requirement.
		{737_626, 69, "one below level 70"},
		{737_627, 70, "level 70 threshold"},
		// Level 85 ≈ 25 % of level 99 XP.
		{3_258_593, 84, "one below level 85"},
		{3_258_594, 85, "level 85 threshold"},
		// Level 92 = exactly 50 % of level 99 XP (wiki milestone).
		{6_517_252, 91, "one below level 92"},
		{6_517_253, 92, "level 92 threshold (half of 99)"},
		// Level 99 — maximum standard OSRS level.
		{13_034_430, 98, "one below level 99"},
		{13_034_431, 99, "level 99 threshold"},
		// Virtual level 100.
		{14_391_159, 99, "one below virtual level 100"},
		{14_391_160, 100, "virtual level 100 threshold"},
		// Virtual level 126 — highest tracked level.
		{188_884_739, 125, "one below virtual level 126"},
		{188_884_740, 126, "virtual level 126 threshold"},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			got := LevelForXP(tc.xp)
			if got != tc.wantLvl {
				t.Errorf("LevelForXP(%d) = %d, want %d", tc.xp, got, tc.wantLvl)
			}
		})
	}
}

func TestLevelForXP_AtXPCap(t *testing.T) {
	// 200,000,000 is the hard XP cap — must return the maximum virtual level.
	if got := LevelForXP(200_000_000); got != 126 {
		t.Errorf("LevelForXP(200_000_000) = %d, want 126", got)
	}
}

func TestLevelForXP_AboveXPCap(t *testing.T) {
	// XP beyond the cap is capped at level 126.
	if got := LevelForXP(200_000_001); got != 126 {
		t.Errorf("LevelForXP(200_000_001) = %d, want 126", got)
	}
}

// ----------------------------------------------------------------------------
// XPRangeForLevel
// ----------------------------------------------------------------------------

func TestXPRangeForLevel_InvalidLevels(t *testing.T) {
	invalids := []int{-1, 0, 127, 200}
	for _, level := range invalids {
		t.Run(itoa(level), func(t *testing.T) {
			min, max := XPRangeForLevel(level)
			if min != -1 || max != -1 {
				t.Errorf("XPRangeForLevel(%d) = (%d, %d), want (-1, -1)", level, min, max)
			}
		})
	}
}

func TestXPRangeForLevel_Level1(t *testing.T) {
	// Level 1: 0 XP (min) to 82 XP (one below the level 2 threshold of 83).
	min, max := XPRangeForLevel(1)
	if min != 0 {
		t.Errorf("XPRangeForLevel(1) minXP = %d, want 0", min)
	}
	if max != 82 {
		t.Errorf("XPRangeForLevel(1) maxXP = %d, want 82", max)
	}
}

func TestXPRangeForLevel_Level2(t *testing.T) {
	// Level 2: 83 XP to 173 XP (one below level 3 threshold of 174).
	min, max := XPRangeForLevel(2)
	if min != 83 {
		t.Errorf("XPRangeForLevel(2) minXP = %d, want 83", min)
	}
	if max != 173 {
		t.Errorf("XPRangeForLevel(2) maxXP = %d, want 173", max)
	}
}

func TestXPRangeForLevel_Level99(t *testing.T) {
	// Level 99: 13,034,431 XP to one below virtual level 100 threshold.
	const wantMin = 13_034_431
	const wantMax = 14_391_160 - 1 // xpTable[100] - 1
	min, max := XPRangeForLevel(99)
	if min != wantMin {
		t.Errorf("XPRangeForLevel(99) minXP = %d, want %d", min, wantMin)
	}
	if max != wantMax {
		t.Errorf("XPRangeForLevel(99) maxXP = %d, want %d", max, wantMax)
	}
}

func TestXPRangeForLevel_Level126(t *testing.T) {
	// Level 126 (max virtual): threshold to the hard XP cap of 200,000,000.
	const wantMin = 188_884_740
	const wantMax = 200_000_000
	min, max := XPRangeForLevel(126)
	if min != wantMin {
		t.Errorf("XPRangeForLevel(126) minXP = %d, want %d", min, wantMin)
	}
	if max != wantMax {
		t.Errorf("XPRangeForLevel(126) maxXP = %d, want %d", max, wantMax)
	}
}

func TestXPRangeForLevel_RangeContainsThreshold(t *testing.T) {
	// For every level 1–126, LevelForXP(minXP) must equal that level, and
	// LevelForXP(maxXP) must also equal that level.
	for level := 1; level <= 126; level++ {
		min, max := XPRangeForLevel(level)
		if gotMin := LevelForXP(min); gotMin != level {
			t.Errorf("LevelForXP(XPRangeForLevel(%d).min=%d) = %d, want %d",
				level, min, gotMin, level)
		}
		if gotMax := LevelForXP(max); gotMax != level {
			t.Errorf("LevelForXP(XPRangeForLevel(%d).max=%d) = %d, want %d",
				level, max, gotMax, level)
		}
	}
}

// ----------------------------------------------------------------------------
// xpForLevelFormula — official OSRS formula vs hardcoded table
// ----------------------------------------------------------------------------

// TestXPForLevelFormula_MatchesTable verifies that the formula implementation
// produces identical values to every entry in the hardcoded xpTable for levels
// 1–126. This guards against copy-paste errors in the table and validates that
// the formula matches the wiki specification exactly.
func TestXPForLevelFormula_MatchesTable(t *testing.T) {
	for level := 1; level <= 126; level++ {
		got := xpForLevelFormula(level)
		want := xpTable[level]
		if got != want {
			t.Errorf("xpForLevelFormula(%d) = %d, want %d (table value)",
				level, got, want)
		}
	}
}

// TestXPForLevelFormula_WikiSpotChecks verifies specific values cited on the
// OSRS wiki experience page as canonical milestones.
func TestXPForLevelFormula_WikiSpotChecks(t *testing.T) {
	cases := []struct {
		level int
		want  int
	}{
		{1, 0},
		{2, 83},
		{10, 1_154},
		{30, 13_363},
		{50, 101_333},
		// Level 85 ≈ 25 % of level 99 XP.
		{85, 3_258_594},
		// Level 92 = exactly 50 % of level 99 XP.
		{92, 6_517_253},
		// Level 99 maximum standard level.
		{99, 13_034_431},
	}

	for _, tc := range cases {
		t.Run(itoa(tc.level), func(t *testing.T) {
			got := xpForLevelFormula(tc.level)
			if got != tc.want {
				t.Errorf("xpForLevelFormula(%d) = %d, want %d", tc.level, got, tc.want)
			}
		})
	}
}
