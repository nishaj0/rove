//go:build windows

package networkwatch

import (
	"context"
	"syscall"

	"golang.org/x/sys/windows"
)

// Watch subscribes to Windows IP interface changes. The protection engine
// still confirms connectivity with the configured provider; this event only
// removes an unnecessary retry delay.
func Watch(ctx context.Context, onChange func()) error {
	callback := syscall.NewCallback(func(_, _, _ uintptr) uintptr {
		if onChange != nil {
			go onChange()
		}
		return 0
	})
	var handle windows.Handle
	if err := windows.NotifyIpInterfaceChange(windows.AF_UNSPEC, callback, nil, false, &handle); err != nil {
		return err
	}
	defer windows.CancelMibChangeNotify2(handle)
	<-ctx.Done()
	return nil
}
