//go:build linux

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/tuiperf"
)

func waitForMarker(events <-chan tuiperf.MarkerEvent, marker string, timeout time.Duration) (time.Time, error) {
	deadline := time.After(timeout)
	for {
		select {
		case ev := <-events:
			if ev.Marker == marker {
				return ev.At, nil
			}
		case <-deadline:
			return time.Time{}, fmt.Errorf("timeout waiting for marker %q", marker)
		}
	}
}

func seedFixture(storeRoot, fixtureID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", "./tools/perfseedwait", "--fixture-id", fixtureID, "--store-root", storeRoot)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("seed fixture %s timed out: %w", fixtureID, ctx.Err())
		}
		return fmt.Errorf("seed fixture %s: %w: %s", fixtureID, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func writeEnvelope(path string, envelope checkEnvelope) error {
	raw, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func shellEscape(v string) string {
	if v == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(v, "'", "'\\''") + "'"
}
