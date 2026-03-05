package handler

import (
	"context"
	"net/http"

	appskill "github.com/johnfarrell/runeplan/application/skill"
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
	"github.com/johnfarrell/runeplan/domain/user"
	templatesskill "github.com/johnfarrell/runeplan/interfaces/templates/skill"
	"github.com/johnfarrell/runeplan/interfaces/templates"
)

// GoalLoader loads goals for a given RSN ID.
type GoalLoader interface {
	ListByRSN(ctx context.Context, rsnID string) ([]domgoal.Goal, error)
}

// CatalogLoader loads catalog goals by ID.
type CatalogLoader interface {
	GetByID(ctx context.Context, id string) (*domcatalog.Goal, error)
}

// SkillsHandler returns the HTMX skills fragment.
func SkillsHandler(goalRepo GoalLoader, catalogRepo CatalogLoader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := user.GetUser(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		rsn := u.ActiveRSN()
		if rsn == nil {
			templates.Render(w, r, http.StatusOK, templatesskill.Grid(nil))
			return
		}

		goals, err := goalRepo.ListByRSN(r.Context(), rsn.ID)
		if err != nil {
			templates.Render(w, r, http.StatusInternalServerError, templates.Error("Failed to load goals"))
			return
		}

		catalogGoals := make([]domcatalog.Goal, len(goals))
		for i, g := range goals {
			if g.CatalogID != nil {
				cg, err := catalogRepo.GetByID(r.Context(), *g.CatalogID)
				if err == nil {
					catalogGoals[i] = *cg
				}
			}
		}

		thresholds := appskill.AggregateThresholds(goals, catalogGoals, rsn.SkillLevels)
		templates.Render(w, r, http.StatusOK, templatesskill.Grid(thresholds))
	}
}
