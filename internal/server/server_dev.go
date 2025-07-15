//go:build dev

package server

import (
	"github.com/MuhamedUsman/letshare/internal/config"
)

func GetPort() int {
	if config.TestFlag {
		return TestHTTPPort
	}
	return DefaultPort
}
