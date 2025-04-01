package domain

import "time"

type FileInfo struct {
	Name string
	Path string
	// file size in bytes
	Size int64
	Type string // MIME type, if not resolved then extension
	// last modified time
	ModTime time.Time
}
