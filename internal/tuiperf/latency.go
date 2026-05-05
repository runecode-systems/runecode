package tuiperf

import (
	"bufio"
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

func WatchMarkers(r io.Reader, markers []string, sink chan<- MarkerEvent) {
	if len(markers) == 0 {
		return
	}
	want := map[string]struct{}{}
	for _, marker := range markers {
		trimmed := strings.TrimSpace(marker)
		if trimmed == "" {
			continue
		}
		want[trimmed] = struct{}{}
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		for marker := range want {
			if strings.Contains(line, marker) {
				sink <- MarkerEvent{Marker: marker, At: time.Now()}
			}
		}
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
