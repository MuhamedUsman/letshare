package domain

type FileInfo struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	AccessID uint32 `json:"accessId"`
	// file size in bytes
	Size int64 `json:"size,omitempty"`
}
