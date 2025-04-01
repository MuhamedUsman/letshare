package domain

import "time"

type FileInfo struct {
	// last modified time
	ModTime time.Time `json:"modTime,omitempty,omitzero"`
	Name    string    `json:"name,omitempty"`
	Path    string    `json:"path,omitempty"`
	Type    string    `json:"type,omitempty"` // MIME type, if not resolved then extension
	// file size in bytes
	Size int64 `json:"size,omitempty"`
}
