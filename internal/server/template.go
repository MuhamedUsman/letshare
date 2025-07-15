package server

import (
	"github.com/MuhamedUsman/letshare/internal/webui"
	"github.com/dustin/go-humanize"
	"html/template"
	"path/filepath"
	"strings"
	"sync"
)

var (
	once sync.Once
	t    *template.Template
)

func getTemplate() *template.Template {
	once.Do(func() {
		funcs := template.FuncMap{
			"fileType":      fileType,
			"humanizeSize":  humanizeSize,
			"trimExtSuffix": trimExtSuffix,
		}
		t = template.Must(template.New("fileIndexes").
			Funcs(funcs).
			ParseFS(webui.Files, "index.tmpl.html"),
		)
	})
	return t
}

func trimExtSuffix(name string) string {
	if name == "" {
		return ""
	}
	ext := filepath.Ext(name)
	trimmed := strings.TrimSuffix(name, ext)
	if trimmed == "" {
		return name // .gitignore, for example
	}
	return trimmed
}

func fileType(name string) (typ string) {
	typ = "unknown"
	if name == "" {
		return
	}
	typ = "file"
	// there is only one ".", files like .gitignore
	if strings.LastIndex(name, ".") == 0 {
		return
	}
	typ = filepath.Ext(name)
	// there is no ".", files like "README"
	if typ == "" {
		typ = "file"
		return
	}
	typ = strings.TrimPrefix(typ, ".") // remove the leading dot
	return
}

func humanizeSize(size int64) string {
	return humanize.Bytes(uint64(size))
}
