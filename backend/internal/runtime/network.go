package runtimecfg

import "fmt"

const (
	LoopbackHost   = "127.0.0.1"
	DefaultAPIPort = 7331
	StatusPath     = "/status"
)

// builds a loopback bind address from the provided port.
func LoopbackAddr(port int) string {
	return fmt.Sprintf("%s:%d", LoopbackHost, normalizePort(port))
}

// builds the local status page url used by tray open actions.
func StatusURL(port int) string {
	return fmt.Sprintf("http://%s%s", LoopbackAddr(port), StatusPath)
}

func normalizePort(port int) int {
	if port <= 0 || port > 65535 {
		return DefaultAPIPort
	}
	return port
}
