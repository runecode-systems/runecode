//go:build !windows

package launcherdaemon

import "os"

func replaceRuntimeStateFile(tempPath string, destPath string) error {
	return os.Rename(tempPath, destPath)
}
