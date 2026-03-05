package hiscores

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/johnfarrell/runeplan/domain/skill"
)

// HiscoreSkillOrder is the canonical order OSRS returns skills in the CSV response.
// Must match the order returned by the hiscore API exactly.
var HiscoreSkillOrder = []skill.Skill{
	skill.Attack, skill.Defence, skill.Strength, skill.Hitpoints,
	skill.Ranged, skill.Prayer, skill.Magic, skill.Cooking,
	skill.Woodcut, skill.Fletching, skill.Fishing, skill.Firemaking,
	skill.Crafting, skill.Smithing, skill.Mining, skill.Herblore,
	skill.Agility, skill.Thieving, skill.Slayer, skill.Farming,
	skill.Runecraft, skill.Hunter, skill.Construction, skill.Sailing,
}

// Client fetches skill data from the OSRS Hiscores API.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient creates a hiscores client. timeout=0 uses a default of 10s.
func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
	}
}

// Fetch retrieves XP values for all skills for the given RSN.
func (c *Client) Fetch(rsn string) (map[skill.Skill]skill.XP, error) {
	url := c.baseURL + "?player=" + rsn
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("hiscores: fetch %q: %w", rsn, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hiscores: fetch %q: HTTP %d", rsn, resp.StatusCode)
	}

	result := make(map[skill.Skill]skill.XP, len(HiscoreSkillOrder))
	scanner := bufio.NewScanner(resp.Body)
	i := 0
	for scanner.Scan() && i < len(HiscoreSkillOrder) {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			i++
			continue
		}
		xpVal, err := strconv.Atoi(parts[2])
		if err != nil || xpVal < 0 {
			i++
			continue
		}
		xp, err := skill.NewXP(xpVal)
		if err != nil {
			i++
			continue
		}
		result[HiscoreSkillOrder[i]] = xp
		i++
	}
	return result, scanner.Err()
}
