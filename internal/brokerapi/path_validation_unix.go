//go:build !windows

package brokerapi

import "os"

func pathEntryIsLinkOrReparse(_ string, info os.FileInfo) (bool, error) {
	return info.Mode()&os.ModeSymlink != 0, nil
}
