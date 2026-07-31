//go:build windows

package runtimepriority

import (
	"os"

	"golang.org/x/sys/windows"
)

func SetGameplayMode(active bool) error {
	process, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION, false, uint32(os.Getpid()))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(process)
	class := uint32(windows.NORMAL_PRIORITY_CLASS)
	if active {
		class = windows.BELOW_NORMAL_PRIORITY_CLASS
	}
	return windows.SetPriorityClass(process, class)
}
