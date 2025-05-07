package file

import "fmt"

// HumanizeSize converts filesize to user-friendly string
//
// Parameters:
//   - bytes: Filesize in bytes
//
// Returns:
//   - string: formated strings of filesize (KB, MB, GB)
func HumanizeSize(bytes uint64) string {
	kb := float64(bytes) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1fKB", kb)
	}
	mb := kb / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.1fMB", mb)
	}
	gb := mb / 1024
	return fmt.Sprintf("%.1fGB", gb)
}
