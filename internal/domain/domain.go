package domain

type FileInfo struct {
	Name     string `json:"name,omitempty"`
	AccessID uint32 `json:"accessId"`
	Size     int64  `json:"size,omitempty"` // Size in bytes
}
