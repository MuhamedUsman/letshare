//go:build !dev

package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetDir returns the user config directory path, if not exists, it creates it.
func GetDir() (string, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config directory look-up: %v", err)
	}
	d = filepath.Join(d, appConfDir)
	// "If path is already a directory, MkdirAll does nothing and returns nil"
	if err = os.MkdirAll(d, 0o750); err != nil {
		return "", fmt.Errorf("creating user config directory: %v", err)
	}
	return d, nil
}
