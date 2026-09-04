package webassets

import (
	"embed"
	"io/fs"
)

//go:embed templates static
var EmbeddedFS embed.FS

// Templates returns the sub-filesystem containing all HTML templates
func Templates() fs.FS {
	sub, err := fs.Sub(EmbeddedFS, "templates")
	if err != nil {
		panic(err)
	}
	return sub
}

// Static returns the sub-filesystem containing all static web assets (js, css, images)
func Static() fs.FS {
	sub, err := fs.Sub(EmbeddedFS, "static")
	if err != nil {
		panic(err)
	}
	return sub
}
