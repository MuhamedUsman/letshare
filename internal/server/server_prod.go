//go:build !dev

package server

func GetPort() int {
	return DefaultPort
}
