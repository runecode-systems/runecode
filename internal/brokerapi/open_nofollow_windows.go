//go:build windows

package brokerapi

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func openReadOnlyNoFollow(path string) (*os.File, error) {
	attrs, err := windows.GetFileAttributes(windows.StringToUTF16Ptr(path))
	if err == nil && attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return nil, fmt.Errorf("open %s: reparse points are not allowed", path)
	}
	h, err := windows.CreateFile(
		windows.StringToUTF16Ptr(path),
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(h), path)
	if f == nil {
		_ = windows.CloseHandle(h)
		return nil, fmt.Errorf("open %s: invalid file handle", path)
	}
	return f, nil
}
