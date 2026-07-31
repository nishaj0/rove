//go:build !windows

package singleinstance

func AcquireDaemon() (func(), bool, error) {
	return func() {}, true, nil
}
