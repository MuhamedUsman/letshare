package util

import (
	"fmt"
)

// UserFriendlyFilesize converts filesize to user-friendly string
//
// Parameters:
//   - bytes: Filesize in bytes
//
// Returns:
//   - string: formated strings of filesize (KB, MB, GB)
func UserFriendlyFilesize(bytes int64) string {
	kb := float64(bytes) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.2fKB", kb)
	}
	mb := kb / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.2fMB", mb)
	}
	gb := mb / 1024
	return fmt.Sprintf("%.2fGB", gb)
}
