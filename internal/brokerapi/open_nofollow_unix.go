//go:build !windows

package brokerapi

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func openReadOnlyNoFollow(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	if f == nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("open %s: invalid file descriptor", path)
	}
	return f, nil
}
