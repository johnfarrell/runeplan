package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/johnfarrell/runeplan/application/catalog"
	domcatalog "github.com/johnfarrell/runeplan/domain/catalog"
	"github.com/johnfarrell/runeplan/domain/goal"
	"github.com/johnfarrell/runeplan/interfaces/handler"
)

type fakeCatalogRepo struct {
	goals []domcatalog.Goal
}

func (f *fakeCatalogRepo) ListAll(ctx context.Context) ([]domcatalog.Goal, error) {
	return f.goals, nil
}
func (f *fakeCatalogRepo) ListByType(ctx context.Context, t goal.Type) ([]domcatalog.Goal, error) {
	var out []domcatalog.Goal
	for _, g := range f.goals {
		if g.Type == t {
			out = append(out, g)
		}
	}
	return out, nil
}
func (f *fakeCatalogRepo) GetByID(ctx context.Context, id string) (*domcatalog.Goal, error) {
	for _, g := range f.goals {
		if g.ID == id {
			return &g, nil
		}
	}
	return nil, catalog.ErrNotFound
}

func TestBrowseHandler_ReturnsPage(t *testing.T) {
	repo := &fakeCatalogRepo{goals: []domcatalog.Goal{
		{ID: "1", Type: goal.TypeQuest, Title: "Dragon Slayer"},
	}}
	svc := catalog.NewService(repo)
	h := handler.BrowseHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/browse?type=quest", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Dragon Slayer") {
		t.Error("expected goal title in response")
	}
}

func TestCatalogDetailHandler_NotFound(t *testing.T) {
	repo := &fakeCatalogRepo{}
	svc := catalog.NewService(repo)
	h := handler.CatalogDetailHandler(svc)

	r := mux.NewRouter()
	r.Handle("/browse/catalog/{id}", h)

	req := httptest.NewRequest(http.MethodGet, "/browse/catalog/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", w.Code)
	}
}
