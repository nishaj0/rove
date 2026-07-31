//go:build windows

package googledrive

import (
	"encoding/base64"
	"unsafe"

	"golang.org/x/sys/windows"
)

func protect(value string) (string, error) {
	bytes := []byte(value)
	in := windows.DataBlob{Size: uint32(len(bytes))}
	if len(bytes) > 0 {
		in.Data = &bytes[0]
	}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return "", err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return base64.StdEncoding.EncodeToString(unsafe.Slice(out.Data, int(out.Size))), nil
}

func unprotect(value string) (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	in := windows.DataBlob{Size: uint32(len(bytes))}
	if len(bytes) > 0 {
		in.Data = &bytes[0]
	}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return "", err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return string(unsafe.Slice(out.Data, int(out.Size))), nil
}
