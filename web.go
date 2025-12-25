// Package discordstayonline provides the embedded web assets for the Discord Stay Online service.
package discordstayonline

import (
	"embed"
	"io/fs"
)

//go:embed web/dist/*
var WebFS embed.FS

// GetWebFS returns the embedded web filesystem with the "web/dist/" prefix stripped.
func GetWebFS() (fs.FS, error) {
	return fs.Sub(WebFS, "web/dist")
}
