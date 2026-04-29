//go:build windows

package launcherdaemon

import (
	"golang.org/x/sys/windows"
)

func replaceRuntimeStateFile(tempPath string, destPath string) error {
	tempUTF16, err := windows.UTF16PtrFromString(tempPath)
	if err != nil {
		return err
	}
	destUTF16, err := windows.UTF16PtrFromString(destPath)
	if err != nil {
		return err
	}
	flags := uint32(windows.MOVEFILE_REPLACE_EXISTING | windows.MOVEFILE_WRITE_THROUGH)
	if err := windows.MoveFileEx(tempUTF16, destUTF16, flags); err != nil {
		return err
	}
	return nil
}
