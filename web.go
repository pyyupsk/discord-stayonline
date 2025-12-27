package discordstayonline

import (
	"embed"
	"io/fs"
)

//go:embed all:web/dist
var WebFS embed.FS

func GetWebFS() (fs.FS, error) {
	return fs.Sub(WebFS, "web/dist")
}
