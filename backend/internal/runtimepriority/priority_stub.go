//go:build !windows

package runtimepriority

func SetGameplayMode(_ bool) error { return nil }
