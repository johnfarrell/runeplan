package handler

import (
	"net/http"

	appuser "github.com/johnfarrell/runeplan/application/user"
	domskill "github.com/johnfarrell/runeplan/domain/skill"
	"github.com/johnfarrell/runeplan/domain/user"
	templatesskill "github.com/johnfarrell/runeplan/interfaces/templates/skill"
	templatesuser "github.com/johnfarrell/runeplan/interfaces/templates/user"
	"github.com/johnfarrell/runeplan/interfaces/templates"
)

// ProfileHandler returns the profile page.
func ProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok := user.GetUser(r.Context())
		if !ok {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		templates.Render(w, r, http.StatusOK, templatesuser.Profile(u))
	}
}

// SyncHandler fetches hiscores for the given RSN and returns the updated skills fragment.
func SyncHandler(svc *appuser.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			templates.Render(w, r, http.StatusBadRequest, templates.Error("Invalid request"))
			return
		}
		rsnID := r.FormValue("rsn_id")
		rsnName := r.FormValue("rsn")
		if rsnID == "" || rsnName == "" {
			templates.Render(w, r, http.StatusBadRequest, templates.Error("Missing rsn_id or rsn"))
			return
		}

		_, err := svc.SyncHiscores(r.Context(), rsnID, rsnName)
		if err != nil {
			templates.Render(w, r, http.StatusBadGateway, templates.Error("Hiscores sync failed: "+err.Error()))
			return
		}

		// Return empty skills fragment — planner will reload via HTMX
		_ = domskill.All
		templates.Render(w, r, http.StatusOK, templatesskill.Grid(nil))
	}
}
