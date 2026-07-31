//go:build windows

package singleinstance

import (
	"errors"

	"golang.org/x/sys/windows"
)

// AcquireDaemon uses the per-session Local namespace, which is naturally
// scoped to the signed-in Windows user/session.
func AcquireDaemon() (func(), bool, error) {
	name, err := windows.UTF16PtrFromString(`Local\CloudSaverDaemon`)
	if err != nil {
		return nil, false, err
	}
	handle, err := windows.CreateMutex(nil, false, name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		if handle != 0 {
			_ = windows.CloseHandle(handle)
		}
		return func() {}, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return func() { _ = windows.CloseHandle(handle) }, true, nil
}
