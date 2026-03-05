package skill

import (
	"errors"
	"testing"
)

// ----------------------------------------------------------------------------
// NewXP
// ----------------------------------------------------------------------------

func TestNewXP_Zero(t *testing.T) {
	xp, err := NewXP(0)
	if err != nil {
		t.Fatalf("NewXP(0) unexpected error: %v", err)
	}
	if xp.Value() != 0 {
		t.Errorf("Value() = %d, want 0", xp.Value())
	}
}

func TestNewXP_Positive(t *testing.T) {
	xp, err := NewXP(83)
	if err != nil {
		t.Fatalf("NewXP(83) unexpected error: %v", err)
	}
	if xp.Value() != 83 {
		t.Errorf("Value() = %d, want 83", xp.Value())
	}
}

func TestNewXP_AtCap(t *testing.T) {
	// 200,000,000 is the OSRS XP cap — must be accepted.
	xp, err := NewXP(200_000_000)
	if err != nil {
		t.Fatalf("NewXP(200_000_000) unexpected error: %v", err)
	}
	if xp.Value() != 200_000_000 {
		t.Errorf("Value() = %d, want 200_000_000", xp.Value())
	}
}

func TestNewXP_Negative(t *testing.T) {
	_, err := NewXP(-1)
	if !errors.Is(err, ErrInvalidXP) {
		t.Errorf("NewXP(-1) error = %v, want ErrInvalidXP", err)
	}
}

// ----------------------------------------------------------------------------
// NewLevel
// ----------------------------------------------------------------------------

func TestNewLevel_MinBoundary(t *testing.T) {
	l, err := NewLevel(1)
	if err != nil {
		t.Fatalf("NewLevel(1) unexpected error: %v", err)
	}
	if l.Value() != 1 {
		t.Errorf("Value() = %d, want 1", l.Value())
	}
}

func TestNewLevel_MaxBoundary(t *testing.T) {
	// 99 is the highest standard (non-virtual) OSRS level.
	l, err := NewLevel(99)
	if err != nil {
		t.Fatalf("NewLevel(99) unexpected error: %v", err)
	}
	if l.Value() != 99 {
		t.Errorf("Value() = %d, want 99", l.Value())
	}
}

func TestNewLevel_Mid(t *testing.T) {
	l, err := NewLevel(50)
	if err != nil {
		t.Fatalf("NewLevel(50) unexpected error: %v", err)
	}
	if l.Value() != 50 {
		t.Errorf("Value() = %d, want 50", l.Value())
	}
}

func TestNewLevel_Zero(t *testing.T) {
	_, err := NewLevel(0)
	if !errors.Is(err, ErrInvalidLevel) {
		t.Errorf("NewLevel(0) error = %v, want ErrInvalidLevel", err)
	}
}

func TestNewLevel_BelowMin(t *testing.T) {
	_, err := NewLevel(-1)
	if !errors.Is(err, ErrInvalidLevel) {
		t.Errorf("NewLevel(-1) error = %v, want ErrInvalidLevel", err)
	}
}

func TestNewLevel_AboveMax(t *testing.T) {
	// Level 127+ is not a valid OSRS standard level.
	_, err := NewLevel(127)
	if !errors.Is(err, ErrInvalidLevel) {
		t.Errorf("NewLevel(127) error = %v, want ErrInvalidLevel", err)
	}
}

// ----------------------------------------------------------------------------
// Level.ToXP — wiki-sourced XP thresholds
// https://oldschool.runescape.wiki/w/Experience
// ----------------------------------------------------------------------------

// levelToXPCases maps standard OSRS levels to their exact XP thresholds as
// published on the wiki. Each entry is the minimum XP required to reach that level.
var levelToXPCases = []struct {
	level int
	xp    int
}{
	{1, 0},
	{2, 83},
	{3, 174},
	{4, 276},
	{5, 388},
	{10, 1_154},
	{20, 4_470},
	{30, 13_363},
	{40, 37_224},
	{50, 101_333},
	// Level 70 — common diary requirement threshold
	{70, 737_627},
	// Level 85 — approximately 25 % of the XP needed for level 99
	{85, 3_258_594},
	// Level 92 — exactly 50 % of the XP needed for level 99 (wiki milestone)
	{92, 6_517_253},
	// Level 99 — maximum standard OSRS level: 13,034,431 XP
	{99, 13_034_431},
}

func TestLevel_ToXP(t *testing.T) {
	for _, tc := range levelToXPCases {
		t.Run(itoa(tc.level), func(t *testing.T) {
			l, err := NewLevel(tc.level)
			if err != nil {
				t.Fatalf("NewLevel(%d) error: %v", tc.level, err)
			}
			got := l.ToXP().Value()
			if got != tc.xp {
				t.Errorf("Level(%d).ToXP() = %d, want %d", tc.level, got, tc.xp)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// XP.ToLevel — derive level from XP using wiki thresholds
// ----------------------------------------------------------------------------

// xpToLevelCases covers boundaries at and around each notable threshold.
var xpToLevelCases = []struct {
	xp    int
	level int
}{
	// Any XP below the level 2 threshold (83) is level 1.
	{0, 1},
	{1, 1},
	{82, 1},
	// Exactly at the level 2 threshold.
	{83, 2},
	{84, 2},
	// One below the level 10 threshold (1,154), and exactly at it.
	{1_153, 9},
	{1_154, 10},
	// One below the level 50 threshold (101,333).
	{101_332, 49},
	{101_333, 50},
	// Level 70 boundary — common diary skill requirement.
	{737_626, 69},
	{737_627, 70},
	// Level 92 = exactly half of level 99 XP per the wiki.
	{6_517_253, 92},
	// One below the level 99 threshold.
	{13_034_430, 98},
	// Exactly at the level 99 threshold.
	{13_034_431, 99},
	// XP beyond level 99 (virtual territory) still returns 99 since Level
	// is bounded to [1, 99].
	{14_391_160, 100},
	{200_000_000, 126},
}

func TestXP_ToLevel(t *testing.T) {
	for _, tc := range xpToLevelCases {
		t.Run(itoa(tc.xp), func(t *testing.T) {
			xp, err := NewXP(tc.xp)
			if err != nil {
				t.Fatalf("NewXP(%d) error: %v", tc.xp, err)
			}
			got := xp.ToLevel().Value()
			if got != tc.level {
				t.Errorf("XP(%d).ToLevel() = %d, want %d", tc.xp, got, tc.level)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// XP.XPRemaining
// ----------------------------------------------------------------------------

func TestXPRemaining_AlreadyAtTarget(t *testing.T) {
	// Exactly at the level 50 threshold — 0 XP remaining.
	xp, _ := NewXP(101_333)
	target, _ := NewLevel(50)
	if got := xp.XPRemaining(target); got != 0 {
		t.Errorf("XPRemaining = %d, want 0", got)
	}
}

func TestXPRemaining_AboveTarget(t *testing.T) {
	// Well past level 50 — 0 XP remaining.
	xp, _ := NewXP(200_000)
	target, _ := NewLevel(50)
	if got := xp.XPRemaining(target); got != 0 {
		t.Errorf("XPRemaining = %d, want 0", got)
	}
}

func TestXPRemaining_FromZeroToLevel2(t *testing.T) {
	// 0 XP → level 2 requires 83 XP.
	xp, _ := NewXP(0)
	target, _ := NewLevel(2)
	if got := xp.XPRemaining(target); got != 83 {
		t.Errorf("XPRemaining = %d, want 83", got)
	}
}

func TestXPRemaining_PartialProgress(t *testing.T) {
	// 100 XP → level 10 requires 1,154 XP, so 1,054 remaining.
	xp, _ := NewXP(100)
	target, _ := NewLevel(10)
	const want = 1_154 - 100
	if got := xp.XPRemaining(target); got != want {
		t.Errorf("XPRemaining = %d, want %d", got, want)
	}
}

func TestXPRemaining_Level92IsHalfOf99(t *testing.T) {
	// Wiki fact: 6,517,253 XP is exactly half of the XP needed for level 99.
	// A player at exactly level 92 has 0 XP remaining to level 92.
	xp, _ := NewXP(6_517_253)
	target, _ := NewLevel(92)
	if got := xp.XPRemaining(target); got != 0 {
		t.Errorf("XPRemaining = %d, want 0", got)
	}
}

func TestXPRemaining_ToMax(t *testing.T) {
	// 0 XP → level 99 requires 13,034,431 XP.
	xp, _ := NewXP(0)
	target, _ := NewLevel(99)
	const want = 13_034_431
	if got := xp.XPRemaining(target); got != want {
		t.Errorf("XPRemaining = %d, want %d", got, want)
	}
}

// itoa converts an int to a string for use as a subtest name without
// importing strconv — keeps the test file dependency-free.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
