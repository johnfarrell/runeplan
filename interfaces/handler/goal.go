package handler

import (
	"net/http"

	"github.com/gorilla/mux"
	goalapp "github.com/johnfarrell/runeplan/application/goal"
	domgoal "github.com/johnfarrell/runeplan/domain/goal"
	"github.com/johnfarrell/runeplan/domain/user"
	"github.com/johnfarrell/runeplan/interfaces/templates"
	templatesgoal "github.com/johnfarrell/runeplan/interfaces/templates/goal"
)

// PlannerHandler returns the full planner page.
func PlannerHandler(svc *goalapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := user.GetUser(r.Context())
		if !ok {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		rsn := u.ActiveRSN()
		if rsn == nil {
			templates.Render(w, r, http.StatusOK, templatesgoal.Planner(nil))
			return
		}
		goals, err := svc.List(r.Context(), rsn.ID)
		if err != nil {
			templates.Render(w, r, http.StatusInternalServerError, templates.Error("Failed to load goals"))
			return
		}
		templates.Render(w, r, http.StatusOK, templatesgoal.Planner(goals))
	}
}

// ActivateGoalHandler activates a catalog goal for the current RSN.
func ActivateGoalHandler(svc *goalapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := user.GetUser(r.Context())
		if !ok {
			w.Header().Set("HX-Redirect", "/")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		rsn := u.ActiveRSN()
		if rsn == nil {
			templates.Render(w, r, http.StatusBadRequest, templates.Error("No RSN linked"))
			return
		}
		if err := r.ParseForm(); err != nil {
			templates.Render(w, r, http.StatusBadRequest, templates.Error("Invalid request"))
			return
		}
		catalogID := r.FormValue("catalog_id")
		g, err := svc.Activate(r.Context(), rsn.ID, catalogID)
		if err != nil {
			templates.Render(w, r, http.StatusInternalServerError, templates.Error("Failed to activate goal"))
			return
		}
		templates.Render(w, r, http.StatusOK, templatesgoal.GoalCard(*g))
	}
}

// CompleteGoalHandler marks a goal as complete and returns the updated card.
func CompleteGoalHandler(svc *goalapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		goalID := mux.Vars(r)["id"]
		if err := svc.Complete(r.Context(), goalID); err != nil {
			templates.Render(w, r, http.StatusInternalServerError, templates.Error("Failed to complete goal"))
			return
		}
		u, _ := user.GetUser(r.Context())
		rsn := u.ActiveRSN()
		if rsn == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		goals, _ := svc.List(r.Context(), rsn.ID)
		for _, g := range goals {
			if g.ID == goalID {
				templates.Render(w, r, http.StatusOK, templatesgoal.GoalCard(g))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}

// ToggleRequirementHandler toggles a requirement and returns the updated row.
func ToggleRequirementHandler(svc *goalapp.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requirementID := mux.Vars(r)["id"]
		if err := r.ParseForm(); err != nil {
			templates.Render(w, r, http.StatusBadRequest, templates.Error("Invalid request"))
			return
		}
		goalID := r.FormValue("goal_id")
		completed, err := svc.ToggleRequirement(r.Context(), goalID, requirementID)
		if err != nil {
			templates.Render(w, r, http.StatusInternalServerError, templates.Error("Failed to toggle"))
			return
		}
		templates.Render(w, r, http.StatusOK, templatesgoal.RequirementRow(domgoal.RequirementProgress{
			GoalID:        goalID,
			RequirementID: requirementID,
			Completed:     completed,
		}))
	}
}
