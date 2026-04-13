//go:build windows

package brokerapi

import (
	"path/filepath"
	"testing"
)

func TestOpenReadOnlyNoFollowRejectsMissingPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing-marker")
	if _, err := openReadOnlyNoFollow(missing); err == nil {
		t.Fatal("openReadOnlyNoFollow expected missing-path error")
	}
}
