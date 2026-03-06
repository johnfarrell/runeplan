package handler

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/johnfarrell/runeplan/application/catalog"
	"github.com/johnfarrell/runeplan/domain/goal"
	"github.com/johnfarrell/runeplan/interfaces/templates"
	templatescatalog "github.com/johnfarrell/runeplan/interfaces/templates/catalog"
)

// BrowseHandler returns the catalog browse page handler.
func BrowseHandler(svc *catalog.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		typeParam := r.URL.Query().Get("type")
		if typeParam == "" {
			typeParam = string(goal.TypeQuest)
		}
		t := goal.Type(typeParam)
		if !t.Valid() {
			t = goal.TypeQuest
		}

		goals, err := svc.ListByType(r.Context(), t)
		if err != nil {
			templates.Render(w, r, http.StatusInternalServerError, templates.Error("Failed to load goals"))
			return
		}

		// HTMX tab swap — return fragment only
		if r.Header.Get("HX-Request") == "true" {
			templates.Render(w, r, http.StatusOK, templatescatalog.GoalList(goals))
			return
		}

		templates.Render(w, r, http.StatusOK, templatescatalog.Browse(t, goals))
	}
}

// CatalogDetailHandler returns the catalog goal detail page handler.
func CatalogDetailHandler(svc *catalog.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		g, err := svc.GetByID(r.Context(), id)
		if err != nil {
			templates.Render(w, r, http.StatusNotFound, templates.Error("Goal not found"))
			return
		}
		templates.Render(w, r, http.StatusOK, templatescatalog.Detail(g))
	}
}
