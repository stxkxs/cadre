package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist/*
var distFS embed.FS

// staticHandler serves the embedded frontend with SPA fallback.
// Any path that doesn't start with /api/ and isn't a real file gets index.html.
func staticHandler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("failed to create sub filesystem: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Skip API routes (shouldn't reach here, but just in case).
		if strings.HasPrefix(path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the file directly.
		if path != "/" {
			cleanPath := strings.TrimPrefix(path, "/")
			if f, err := sub.Open(cleanPath); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// SPA fallback: serve index.html for all other routes.
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
