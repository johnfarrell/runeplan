package skill

import "math"

// xpTable holds the minimum XP required for each level (index = level, 1-indexed).
// Levels 1–126 are supported, plus a sentinel for the XP cap (200,000,000).
// Level 1 always starts at 0 XP.
// Virtual levels 100–126 are included per the OSRS wiki.
var xpTable = [...]int{
	0,        // placeholder so index == level
	0,        // Level 1
	83,       // Level 2
	174,      // Level 3
	276,      // Level 4
	388,      // Level 5
	512,      // Level 6
	650,      // Level 7
	801,      // Level 8
	969,      // Level 9
	1154,     // Level 10
	1358,     // Level 11
	1584,     // Level 12
	1833,     // Level 13
	2107,     // Level 14
	2411,     // Level 15
	2746,     // Level 16
	3115,     // Level 17
	3523,     // Level 18
	3973,     // Level 19
	4470,     // Level 20
	5018,     // Level 21
	5624,     // Level 22
	6291,     // Level 23
	7028,     // Level 24
	7842,     // Level 25
	8740,     // Level 26
	9730,     // Level 27
	10824,    // Level 28
	12031,    // Level 29
	13363,    // Level 30
	14833,    // Level 31
	16456,    // Level 32
	18247,    // Level 33
	20224,    // Level 34
	22406,    // Level 35
	24815,    // Level 36
	27473,    // Level 37
	30408,    // Level 38
	33648,    // Level 39
	37224,    // Level 40
	41171,    // Level 41
	45529,    // Level 42
	50339,    // Level 43
	55649,    // Level 44
	61512,    // Level 45
	67983,    // Level 46
	75127,    // Level 47
	83014,    // Level 48
	91721,    // Level 49
	101333,   // Level 50
	111945,   // Level 51
	123660,   // Level 52
	136594,   // Level 53
	150872,   // Level 54
	166636,   // Level 55
	184040,   // Level 56
	203254,   // Level 57
	224466,   // Level 58
	247886,   // Level 59
	273742,   // Level 60
	302288,   // Level 61
	333804,   // Level 62
	368599,   // Level 63
	407015,   // Level 64
	449428,   // Level 65
	496254,   // Level 66
	547953,   // Level 67
	605032,   // Level 68
	668051,   // Level 69
	737627,   // Level 70
	814445,   // Level 71
	899257,   // Level 72
	992895,   // Level 73
	1096278,  // Level 74
	1210421,  // Level 75
	1336443,  // Level 76
	1475581,  // Level 77
	1629200,  // Level 78
	1798808,  // Level 79
	1986068,  // Level 80
	2192818,  // Level 81
	2421087,  // Level 82
	2673114,  // Level 83
	2951373,  // Level 84
	3258594,  // Level 85
	3597792,  // Level 86
	3972294,  // Level 87
	4385776,  // Level 88
	4842295,  // Level 89
	5346332,  // Level 90
	5902831,  // Level 91
	6517253,  // Level 92
	7195629,  // Level 93
	7944614,  // Level 94
	8771558,  // Level 95
	9684577,  // Level 96
	10692629, // Level 97
	11805606, // Level 98
	13034431, // Level 99
	// Virtual levels 100–126
	14391160,  // Level 100
	15889109,  // Level 101
	17542976,  // Level 102
	19368992,  // Level 103
	21385073,  // Level 104
	23611006,  // Level 105
	26068632,  // Level 106
	28782069,  // Level 107
	31777943,  // Level 108
	35085654,  // Level 109
	38737661,  // Level 110
	42769801,  // Level 111
	47221641,  // Level 112
	52136869,  // Level 113
	57563718,  // Level 114
	63555443,  // Level 115
	70170840,  // Level 116
	77474828,  // Level 117
	85539082,  // Level 118
	94442737,  // Level 119
	104273167, // Level 120
	115126838, // Level 121
	127110260, // Level 122
	140341028, // Level 123
	154948977, // Level 124
	171077457, // Level 125
	188884740, // Level 126
}

// maxXP is the hard cap for XP in any skill (200 million).
const maxXP = 200_000_000

// maxLevel is the highest virtual level tracked in the table.
const maxLevel = 126

// LevelForXP returns the (virtual) level corresponding to a given XP value.
// Returns 1 for XP < 83, and caps at 126 for XP ≥ 188,884,740
// (beyond which the skill is maxed but no further levels are tracked).
func LevelForXP(xp int) int {
	if xp < 0 {
		xp = 0
	}
	if xp >= maxXP {
		// XP is at or beyond the cap; return the highest virtual level.
		return maxLevel
	}
	// Binary search for the highest level whose XP requirement is ≤ xp.
	lo, hi := 1, maxLevel
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if xpTable[mid] <= xp {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// XPRangeForLevel returns the minimum and maximum XP values (inclusive) that
// correspond to the given level.
//
//   - minXP: the XP at which the player enters this level.
//   - maxXP: one less than the XP needed for the next level (or the skill XP
//     cap of 200,000,000 for level 126).
//
// Returns (-1, -1) for any level outside the range [1, 126].
func XPRangeForLevel(level int) (minXP, maxXP int) {
	if level < 1 || level > maxLevel {
		return -1, -1
	}
	minXP = xpTable[level]
	if level == maxLevel {
		maxXP = 200_000_000
	} else {
		maxXP = xpTable[level+1] - 1
	}
	return minXP, maxXP
}

// xpForLevelFormula computes the minimum XP for a level using the official
// OSRS formula:  floor( (1/4) * sum_{l=1}^{L-1} floor(l + 300 * 2^(l/7)) )
// This matches the wiki table exactly and can be used to verify or extend the
// table beyond level 126.
func xpForLevelFormula(level int) int {
	total := 0.0
	for l := 1; l < level; l++ {
		total += math.Floor(float64(l) + 300.0*math.Pow(2.0, float64(l)/7.0))
	}
	return int(math.Floor(total / 4.0))
}
