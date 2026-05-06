package tuiperf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type MarkerEvent struct {
	Marker string
	At     time.Time
}

func WatchMarkers(ctx context.Context, r io.Reader, markers []string, sink chan<- MarkerEvent) {
	defer close(sink)
	want := markerSet(markers)
	if len(want) == 0 {
		return
	}
	ctx = normalizeMarkerContext(ctx)
	done := watchMarkerCancellation(ctx, r)
	defer close(done)
	maxMarkerLen := longestMarkerLength(want)
	buffer := make([]byte, 0, maxMarkerLen*2)
	chunk := make([]byte, 4096)
	for {
		n, err := r.Read(chunk)
		updated, stop := processMarkerChunk(ctx, sink, buffer, chunk[:n], want, maxMarkerLen)
		buffer = updated
		if stop {
			return
		}
		if err != nil {
			return
		}
		if markerContextDone(ctx) {
			return
		}
	}
}

func markerSet(markers []string) map[string]struct{} {
	want := map[string]struct{}{}
	for _, marker := range markers {
		trimmed := strings.TrimSpace(marker)
		if trimmed == "" {
			continue
		}
		want[trimmed] = struct{}{}
	}
	return want
}

func longestMarkerLength(want map[string]struct{}) int {
	longest := 0
	for marker := range want {
		if len(marker) > longest {
			longest = len(marker)
		}
	}
	if longest == 0 {
		return 1
	}
	return longest
}

func normalizeMarkerContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func watchMarkerCancellation(ctx context.Context, r io.Reader) chan struct{} {
	done := make(chan struct{})
	closer, ok := r.(io.ReadCloser)
	if !ok {
		return done
	}
	go func() {
		select {
		case <-ctx.Done():
			_ = closer.Close()
		case <-done:
		}
	}()
	return done
}

func emitMarkerMatches(ctx context.Context, sink chan<- MarkerEvent, buffer []byte, want map[string]struct{}) ([]byte, bool) {
	for {
		marker, end, found := nextMarkerMatch(buffer, want)
		if !found {
			return buffer, false
		}
		select {
		case sink <- MarkerEvent{Marker: marker, At: time.Now()}:
			buffer = append([]byte(nil), buffer[end:]...)
		case <-ctx.Done():
			return buffer, true
		}
	}
}

func nextMarkerMatch(buffer []byte, want map[string]struct{}) (string, int, bool) {
	bestIndex := -1
	bestEnd := -1
	bestMarker := ""
	for marker := range want {
		idx := bytes.Index(buffer, []byte(marker))
		if idx < 0 {
			continue
		}
		if bestIndex == -1 || idx < bestIndex {
			bestIndex = idx
			bestEnd = idx + len(marker)
			bestMarker = marker
		}
	}
	if bestIndex == -1 {
		return "", 0, false
	}
	return bestMarker, bestEnd, true
}

func truncateMarkerBuffer(buffer []byte, maxMarkerLen int) []byte {
	keep := maxMarkerLen - 1
	if keep < 1 {
		keep = 1
	}
	if len(buffer) <= keep {
		return buffer
	}
	return append([]byte(nil), buffer[len(buffer)-keep:]...)
}

func processMarkerChunk(
	ctx context.Context,
	sink chan<- MarkerEvent,
	buffer []byte,
	chunk []byte,
	want map[string]struct{},
	maxMarkerLen int,
) ([]byte, bool) {
	if len(chunk) == 0 {
		return buffer, false
	}
	buffer = append(buffer, chunk...)
	updated, stop := emitMarkerMatches(ctx, sink, buffer, want)
	if stop {
		return updated, true
	}
	return truncateMarkerBuffer(updated, maxMarkerLen), false
}

func markerContextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func P95Millis(samples []float64) (float64, error) {
	if len(samples) == 0 {
		return 0, fmt.Errorf("samples required")
	}
	vals := append([]float64(nil), samples...)
	sort.Float64s(vals)
	idx := int(float64(len(vals)-1) * 0.95)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(vals) {
		idx = len(vals) - 1
	}
	return vals[idx], nil
}
