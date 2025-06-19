package domain

type FileInfo struct {
	// last modified time
	AccessID string `json:"accessId"`
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	// file size in bytes
	Size int64 `json:"size,omitempty"`
}

func PopulateFileInfo(filepath string, size int64) FileInfo {
	return FileInfo{}
}
