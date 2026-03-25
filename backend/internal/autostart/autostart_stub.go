//go:build !windows

package autostart

func SetEnabled(_ string, _ string, _ bool) error { return nil }

func IsEnabled(_ string) (bool, error) { return false, nil }
