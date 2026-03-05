package templates

import (
	"net/http"

	"github.com/a-h/templ"
)

// Render writes a templ component to the response with the given status code.
func Render(w http.ResponseWriter, r *http.Request, status int, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = c.Render(r.Context(), w)
}
