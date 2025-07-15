package webui

import (
	"embed"
)

//go:embed "index.tmpl.html" "static"
var Files embed.FS
