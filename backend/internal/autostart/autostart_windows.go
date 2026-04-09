//go:build windows

package autostart

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const runKeyPath = `Software\\Microsoft\\Windows\\CurrentVersion\\Run`

func SetEnabled(appName, exePath string, enabled bool) error {
	if appName == "" {
		return errors.New("appName is required")
	}

	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if enabled {
		command := strings.TrimSpace(exePath)
		if command == "" {
			return errors.New("exePath is required")
		}
		if !strings.HasPrefix(command, "\"") {
			command = fmt.Sprintf("\"%s\"", command)
		}
		return k.SetStringValue(appName, command)
	}
	if err := k.DeleteValue(appName); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return nil
}

func IsEnabled(appName string) (bool, error) {
	if appName == "" {
		return false, errors.New("appName is required")
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer k.Close()

	_, _, err = k.GetStringValue(appName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
