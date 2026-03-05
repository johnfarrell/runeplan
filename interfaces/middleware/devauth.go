package middleware

import (
	"net/http"

	"github.com/johnfarrell/runeplan/domain/skill"
	"github.com/johnfarrell/runeplan/domain/user"
)

// DevAuth injects a hardcoded user into every request context.
// Replace with real session middleware when auth is implemented.
func DevAuth(next http.Handler) http.Handler {
	hardcodedUser := user.User{
		ID: "00000000-0000-0000-0000-000000000001",
		RSNs: []user.RSN{
			{
				ID:          "00000000-0000-0000-0000-000000000002",
				UserID:      "00000000-0000-0000-0000-000000000001",
				RSN:         "Zezima",
				SkillLevels: make(map[skill.Skill]skill.XP),
			},
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := user.SetUser(r.Context(), hardcodedUser)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
