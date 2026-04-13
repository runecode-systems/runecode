//go:build windows

package brokerapi

import "os"

func openReadOnlyNoFollow(path string) (*os.File, error) {
	return os.Open(path)
}
