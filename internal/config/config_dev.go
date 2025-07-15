//go:build dev

package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const testConfDir = ".letshare-test"

var TestFlag bool

func init() {
	flag.BoolVar(&TestFlag,
		"test",
		false,
		"Use to run app with separate user config file & server at port 8080, useful for testing purposes",
	)
	flag.Parse()
}

// GetDir returns the user config directory path, if not exists, it creates it.
func GetDir() (string, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config directory look-up: %v", err)
	}
	if TestFlag {
		d = filepath.Join(d, testConfDir)
	} else {
		d = filepath.Join(d, appConfDir)
	}
	// "If path is already a directory, MkdirAll does nothing and returns nil"
	if err = os.MkdirAll(d, 0o750); err != nil {
		return "", fmt.Errorf("creating user config directory: %v", err)
	}
	return d, nil
}
