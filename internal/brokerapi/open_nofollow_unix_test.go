//go:build !windows

package brokerapi

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestOpenReadOnlyNoFollowUsesCloexecFlag(t *testing.T) {
	if unix.O_CLOEXEC == 0 {
		t.Fatal("unix.O_CLOEXEC = 0, want non-zero close-on-exec flag")
	}
}
