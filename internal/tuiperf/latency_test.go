package tuiperf

import (
	"context"
	"io"
	"runtime"
	"testing"
	"time"
)

func TestP95Millis(t *testing.T) {
	v, err := P95Millis([]float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100})
	if err != nil {
		t.Fatalf("P95Millis returned error: %v", err)
	}
	if v != 90 {
		t.Fatalf("p95 = %.2f, want 90", v)
	}
}

func TestP95MillisRejectsEmpty(t *testing.T) {
	if _, err := P95Millis(nil); err == nil {
		t.Fatal("P95Millis error = nil, want error")
	}
}

func TestWatchMarkersClosesSinkWhenReaderEnds(t *testing.T) {
	t.Parallel()

	r, w := io.Pipe()
	events := make(chan MarkerEvent, 1)
	go WatchMarkers(context.Background(), r, []string{"ready"}, events)
	if _, err := io.WriteString(w, "ready\n"); err != nil {
		t.Fatalf("WriteString error = %v", err)
	}
	_ = w.Close()
	if _, ok := <-events; !ok {
		t.Fatal("events closed before receiving marker")
	}
	if _, ok := <-events; ok {
		t.Fatal("events channel still open, want closed")
	}
}

func TestWatchMarkersReturnsOnCancellation(t *testing.T) {
	t.Parallel()

	r, _ := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan MarkerEvent, 1)
	go WatchMarkers(ctx, r, []string{"ready"}, events)
	cancel()
	select {
	case _, ok := <-events:
		if ok {
			t.Fatal("events channel open after cancellation")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("WatchMarkers did not stop after cancellation")
	}
}

func TestWatchMarkersCancellationHelperExitsAfterEOF(t *testing.T) {
	t.Parallel()
	baseline := runtime.NumGoroutine()
	r, w := io.Pipe()
	events := make(chan MarkerEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go WatchMarkers(ctx, r, []string{"ready"}, events)
	if _, err := io.WriteString(w, "ready\n"); err != nil {
		t.Fatalf("WriteString error = %v", err)
	}
	_ = w.Close()
	for range events {
	}
	for i := 0; i < 20; i++ {
		if runtime.NumGoroutine() <= baseline+1 {
			cancel()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	t.Fatalf("goroutine count stayed elevated: baseline=%d current=%d", baseline, runtime.NumGoroutine())
}
