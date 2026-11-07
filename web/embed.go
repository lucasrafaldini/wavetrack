// Package web holds the dashboard, compiled into the binary.
package web

import "embed"

//go:embed index.html join.html assets
var FS embed.FS
