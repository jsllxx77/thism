package frontend

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var distFS embed.FS

// Handler returns an HTTP handler that serves the embedded React dist.
// It falls back to index.html for any path not found (SPA routing).
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	return &spaHandler{fs: http.FS(sub)}
}

// spaHandler serves static files, falling back to index.html for unknown paths.
type spaHandler struct {
	fs http.FileSystem
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	file, err := h.fs.Open(r.URL.Path)
	if err != nil {
		route := *r
		route.URL.Path = "/"
		http.FileServer(h.fs).ServeHTTP(w, &route)
		return
	}
	_ = file.Close()
	http.FileServer(h.fs).ServeHTTP(w, r)
}
