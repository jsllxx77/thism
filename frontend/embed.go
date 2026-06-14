package frontend

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
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

type SkinRegistry interface {
	Handler(id string) (http.Handler, bool)
}

type SelectedSkinProvider interface {
	GetFrontendSkinID() (string, error)
}

func HandlerWithSkins(classic http.Handler, registry SkinRegistry, selected SelectedSkinProvider) http.Handler {
	if classic == nil {
		classic = Handler()
	}
	return &skinSwitchingHandler{
		classic:  classic,
		registry: registry,
		selected: selected,
	}
}

type skinSwitchingHandler struct {
	classic  http.Handler
	registry SkinRegistry
	selected SelectedSkinProvider
}

func (h *skinSwitchingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.registry != nil && h.selected != nil {
		if id, err := h.selected.GetFrontendSkinID(); err == nil {
			id = strings.TrimSpace(id)
			if id != "" && id != "classic" {
				if handler, ok := h.registry.Handler(id); ok {
					handler.ServeHTTP(w, r)
					return
				}
			}
		}
	}
	h.classic.ServeHTTP(w, r)
}

// spaHandler serves static files, falling back to index.html for unknown paths.
type spaHandler struct {
	fs http.FileSystem
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try to open the requested file
	f, err := h.fs.Open(r.URL.Path)
	if err != nil {
		// Fall back to index.html for SPA routing
		r2 := *r
		r2.URL.Path = "/"
		http.FileServer(h.fs).ServeHTTP(w, &r2)
		return
	}
	f.Close()
	http.FileServer(h.fs).ServeHTTP(w, r)
}
