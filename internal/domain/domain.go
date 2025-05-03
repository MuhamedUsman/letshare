package domain

import (
	"time"
)

type FileInfo struct {
	// last modified time
	ModTime  time.Time `json:"modTime,omitempty,omitzero"`
	AccessID string    `json:"accessId"`
	Name     string    `json:"name,omitempty"`
	Path     string    `json:"-"`
	Type     string    `json:"type,omitempty"`
	// file size in bytes
	Size int64 `json:"size,omitempty"`
}

func PopulateFileInfo(filepath string, size int64) FileInfo {
	return FileInfo{}
}
