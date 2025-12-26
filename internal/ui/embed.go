// Package ui provides static asset serving for the web interface.
// Note: The embed directive cannot embed files from parent directories.
// Static files are embedded at the main package level and passed to handlers.
package ui

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// SPAHandler returns an HTTP handler for serving an SPA with fallback to index.html.
// This ensures client-side routing works by serving index.html for any path that
// doesn't match a static file.
func SPAHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path

		// Clean the path
		if urlPath == "" || urlPath == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Get the file path without leading slash
		filePath := strings.TrimPrefix(urlPath, "/")

		// Check if the path has a file extension (static asset)
		ext := path.Ext(filePath)
		if ext != "" {
			// It's a static asset request - try to serve it
			f, err := fsys.Open(filePath)
			if err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
			// Static asset not found - return 404
			http.NotFound(w, r)
			return
		}

		// No file extension - this is an SPA route, serve index.html
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
