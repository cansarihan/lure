package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui
var assets embed.FS

// uiHandler serves the embedded panel. Embedded files carry no modification
// time, so browsers would otherwise cache them heuristically and keep showing
// an old panel after an upgrade; no-cache makes them revalidate.
func uiHandler() http.Handler {
	sub, err := fs.Sub(assets, "ui")
	if err != nil {
		return http.NotFoundHandler()
	}
	files := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	})
}
