package brokerapi

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

type runtimeMem struct{ allocBytes uint64 }

func (m *runtimeMem) capture() {
	var ms runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&ms)
	m.allocBytes = ms.Alloc
}

func (m runtimeMem) allocMB() float64 {
	return float64(m.allocBytes) / (1024.0 * 1024.0)
}

type phase5CountingFetcher struct {
	payload string
	calls   atomic.Int64
}

func (f *phase5CountingFetcher) Fetch(_ context.Context, _ DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (io.ReadCloser, dependencyRegistryFetchMetadata, error) {
	if lease == nil {
		return nil, dependencyRegistryFetchMetadata{}, fmt.Errorf("auth lease required")
	}
	f.calls.Add(1)
	payload := f.payload
	if payload == "" {
		payload = "phase5-default-payload"
	}
	return io.NopCloser(strings.NewReader(payload)), dependencyRegistryFetchMetadata{ContentType: "application/octet-stream", ExpectedPayloadDigest: artifacts.DigestBytes([]byte(payload))}, nil
}

type phase5GatedFetcher struct {
	gate    chan struct{}
	started chan struct{}
	once    sync.Once
	calls   atomic.Int64
}

func (f *phase5GatedFetcher) Fetch(_ context.Context, _ DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (io.ReadCloser, dependencyRegistryFetchMetadata, error) {
	if lease == nil {
		return nil, dependencyRegistryFetchMetadata{}, fmt.Errorf("auth lease required")
	}
	f.calls.Add(1)
	f.once.Do(func() { close(f.started) })
	<-f.gate
	payload := "phase5-coalesced-payload"
	return io.NopCloser(strings.NewReader(payload)), dependencyRegistryFetchMetadata{ContentType: "application/octet-stream", ExpectedPayloadDigest: artifacts.DigestBytes([]byte(payload))}, nil
}

type phase5ChunkBoundFetcher struct {
	payloadSize int64
	maxReadBuf  int
	maxSeenBuf  atomic.Int64
	readCalls   atomic.Int64
}

func (f *phase5ChunkBoundFetcher) Fetch(_ context.Context, _ DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (io.ReadCloser, dependencyRegistryFetchMetadata, error) {
	if lease == nil {
		return nil, dependencyRegistryFetchMetadata{}, fmt.Errorf("auth lease required")
	}
	reader := &phase5ChunkReader{remaining: f.payloadSize, maxReadBuf: f.maxReadBuf, maxSeenBuf: &f.maxSeenBuf, readCalls: &f.readCalls}
	return io.NopCloser(reader), dependencyRegistryFetchMetadata{ContentType: "application/octet-stream"}, nil
}

type phase5ChunkReader struct {
	remaining  int64
	maxReadBuf int
	maxSeenBuf *atomic.Int64
	readCalls  *atomic.Int64
}

func (r *phase5ChunkReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, io.EOF
	}
	if len(p) > r.maxReadBuf {
		return 0, fmt.Errorf("oversized read buffer")
	}
	r.phase5RememberSeenBuffer(int64(len(p)))
	r.readCalls.Add(1)
	n := len(p)
	if int64(n) > r.remaining {
		n = int(r.remaining)
	}
	for i := 0; i < n; i++ {
		p[i] = 'x'
	}
	r.remaining -= int64(n)
	return n, nil
}

func (r *phase5ChunkReader) phase5RememberSeenBuffer(seen int64) {
	for {
		max := r.maxSeenBuf.Load()
		if seen <= max || r.maxSeenBuf.CompareAndSwap(max, seen) {
			return
		}
	}
}
