package util

import (
	"mime"
	"path"
	"strings"
)

func GetFileType(filename string) string {
	ext := path.Ext(filename)
	fileType := mime.TypeByExtension(ext)
	if fileType == "" {
		fileType = strings.TrimPrefix(ext, ".")
	}
	return fileType
}
