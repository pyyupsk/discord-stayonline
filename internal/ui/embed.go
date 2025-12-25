// Package ui provides static asset serving for the web interface.
// Note: The embed directive cannot embed files from parent directories.
// Static files are embedded at the main package level and passed to handlers.
package ui

import (
	"io/fs"
	"net/http"
)

// StaticHandler returns an HTTP handler for serving static files from the given filesystem.
func StaticHandler(fsys fs.FS) http.Handler {
	return http.FileServer(http.FS(fsys))
}

// StaticHandlerWithPrefix returns a handler that strips a prefix and serves static files.
func StaticHandlerWithPrefix(prefix string, fsys fs.FS) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.FS(fsys)))
}
