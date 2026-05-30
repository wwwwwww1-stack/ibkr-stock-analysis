package priceaction

import (
	"embed"
	"io/fs"
)

//go:embed *.md
var embeddedFS embed.FS

func FS() fs.FS {
	return embeddedFS
}
