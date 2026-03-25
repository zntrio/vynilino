// Package web holds the embedded SPA static assets produced by the UI build.
// The dist/ directory is populated by running `make ui-build`.
package web

import (
	"embed"
	"io/fs"
	"log/slog"
)

//go:embed all:dist
var staticFiles embed.FS

// StaticFS returns the dist/ sub-filesystem for use by the UI HTTP handler.
func StaticFS() fs.FS {
	sub, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		slog.Error("failed to create static sub-FS", "err", err)
		panic(err)
	}
	return sub
}
