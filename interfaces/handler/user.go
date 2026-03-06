package handler

import (
	"fmt"
	"net/http"

	appuser "github.com/johnfarrell/runeplan/application/user"
	"github.com/johnfarrell/runeplan/domain/user"
	"github.com/johnfarrell/runeplan/interfaces/templates"
	templatesuser "github.com/johnfarrell/runeplan/interfaces/templates/user"
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

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `<span class="text-green-400">Synced!</span>`)
	}
}
