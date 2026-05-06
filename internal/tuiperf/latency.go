package tuiperf

import (
	"bufio"
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
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if emitMarkerMatches(ctx, sink, scanner.Text(), want) {
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

func emitMarkerMatches(ctx context.Context, sink chan<- MarkerEvent, line string, want map[string]struct{}) bool {
	for marker := range want {
		if !strings.Contains(line, marker) {
			continue
		}
		select {
		case sink <- MarkerEvent{Marker: marker, At: time.Now()}:
		case <-ctx.Done():
			return true
		}
	}
	return false
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
