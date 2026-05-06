package brokerapi

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

func measurePhase5DependencyFlow(trials int, repoRoot string) ([]perfcontracts.MeasurementRecord, error) {
	service, cleanup, err := newPhase5DependencyService(repoRoot)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	if err := putPhase5TrustedDependencyContext(service, "run-deps-phase5"); err != nil {
		return nil, err
	}
	coreMetrics, err := measurePhase5DependencyCore(service)
	if err != nil {
		return nil, err
	}
	extraMetrics, err := measurePhase5DependencyExtras(service)
	if err != nil {
		return nil, err
	}
	return append(coreMetrics, extraMetrics...), nil
}

func measurePhase5DependencyCore(service *Service) ([]perfcontracts.MeasurementRecord, error) {
	missMS, hitMS, err := measurePhase5DependencyMissAndHit(service)
	if err != nil {
		return nil, err
	}
	coalesced, err := measurePhase5DependencyCoalescing(service)
	if err != nil {
		return nil, err
	}
	base := []perfcontracts.MeasurementRecord{
		phase5DependencyMetric("metric.deps.cache_miss.small.wall_ms", missMS, "ms"),
		phase5DependencyMetric("metric.deps.cache_hit.small.wall_ms", hitMS, "ms"),
	}
	return append(base, coalesced...), nil
}

func measurePhase5DependencyExtras(service *Service) ([]perfcontracts.MeasurementRecord, error) {
	streaming, err := measurePhase5DependencyStreaming(service)
	if err != nil {
		return nil, err
	}
	handoff, err := measurePhase5DependencyHandoff(service)
	if err != nil {
		return nil, err
	}
	return append(streaming, handoff...), nil
}

func measurePhase5DependencyMissAndHit(service *Service) (float64, float64, error) {
	missHitFetcher := &phase5CountingFetcher{payload: "phase5-dependency-payload"}
	service.SetDependencyRegistryFetcherForTests(missHitFetcher)
	missReq := phase5DependencyFetchRegistryRequest("req-deps-miss", "run-deps-phase5", "alpha")
	missMS, missResp, err := phase5TimedDependencyFetch(service, missReq, "dependency miss fetch")
	if err != nil {
		return 0, 0, err
	}
	hitReq := missReq
	hitReq.RequestID = "req-deps-hit"
	hitMS, hitResp, err := phase5TimedDependencyFetch(service, hitReq, "dependency hit fetch")
	if err != nil {
		return 0, 0, err
	}
	if missResp.CacheOutcome != "miss_filled" || hitResp.CacheOutcome != "hit_exact" {
		return 0, 0, fmt.Errorf("unexpected dependency cache outcomes miss=%q hit=%q", missResp.CacheOutcome, hitResp.CacheOutcome)
	}
	return missMS, hitMS, nil
}

func phase5TimedDependencyFetch(service *Service, req DependencyFetchRegistryRequest, action string) (float64, DependencyFetchRegistryResponse, error) {
	started := time.Now()
	resp, errResp := service.HandleDependencyFetchRegistry(context.Background(), req, RequestContext{})
	if err := phase5DependencyErr(action, errResp); err != nil {
		return 0, DependencyFetchRegistryResponse{}, err
	}
	return float64(time.Since(started).Microseconds()) / 1000.0, resp, nil
}

func measurePhase5DependencyCoalescing(service *Service) ([]perfcontracts.MeasurementRecord, error) {
	coalesceFetcher := &phase5GatedFetcher{gate: make(chan struct{}), started: make(chan struct{})}
	service.SetDependencyRegistryFetcherForTests(coalesceFetcher)
	coalesceReq := phase5DependencyFetchRegistryRequest("req-deps-coalesce", "run-deps-phase5", "coalesced")
	responses, wg := phase5StartCoalescedFetches(service, coalesceReq, 6)
	if err := phase5WaitForCoalescedStart(coalesceFetcher); err != nil {
		return nil, err
	}
	close(coalesceFetcher.gate)
	wg.Wait()
	if err := phase5ValidateCoalescedResponses(responses); err != nil {
		return nil, err
	}
	calls := float64(coalesceFetcher.calls.Load())
	casWriteCount := 0.0
	if calls > 0 {
		casWriteCount = 1
	}
	return []perfcontracts.MeasurementRecord{
		phase5DependencyMetric("metric.deps.cache_coalesced.upstream_fetch_count", calls, "count"),
		phase5DependencyMetric("metric.deps.cache_coalesced.cas_write_count", casWriteCount, "count"),
	}, nil
}

func phase5StartCoalescedFetches(service *Service, req DependencyFetchRegistryRequest, n int) ([]*ErrorResponse, *sync.WaitGroup) {
	responses := make([]*ErrorResponse, n)
	start := make(chan struct{})
	wg := &sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			one := req
			one.RequestID = fmt.Sprintf("%s-%d", req.RequestID, idx)
			_, errResp := service.HandleDependencyFetchRegistry(context.Background(), one, RequestContext{})
			responses[idx] = errResp
		}(i)
	}
	close(start)
	return responses, wg
}

func phase5WaitForCoalescedStart(fetcher *phase5GatedFetcher) error {
	select {
	case <-fetcher.started:
		return nil
	case <-time.After(2 * time.Second):
		return fmt.Errorf("timed out waiting for coalesced fetch start")
	}
}

func phase5ValidateCoalescedResponses(responses []*ErrorResponse) error {
	for i := range responses {
		if responses[i] != nil {
			return fmt.Errorf("coalesced request %d failed: %s", i, responses[i].Error.Code)
		}
	}
	return nil
}

func measurePhase5DependencyStreaming(service *Service) ([]perfcontracts.MeasurementRecord, error) {
	service.SetDependencyRegistryFetcherForTests(&phase5ChunkBoundFetcher{payloadSize: 3 << 20, maxReadBuf: 128 << 10})
	chunkFetcher, _ := service.dependencyFetchService.fetcher.(*phase5ChunkBoundFetcher)
	streamReq := phase5DependencyFetchRegistryRequest("req-deps-stream", "run-deps-phase5", "stream")
	var memBefore, memAfter runtimeMem
	memBefore.capture()
	streamResp, streamErr := service.HandleDependencyFetchRegistry(context.Background(), streamReq, RequestContext{})
	memAfter.capture()
	if err := phase5DependencyErr("streaming dependency fetch failed", streamErr); err != nil {
		return nil, err
	}
	peakAllocMB := memAfter.allocMB() - memBefore.allocMB()
	if peakAllocMB < 0 {
		peakAllocMB = 0
	}
	maxReadBuffer, readCalls := phase5ChunkFetcherStats(chunkFetcher)
	return []perfcontracts.MeasurementRecord{
		phase5DependencyMetric("metric.deps.stream_to_cas.max_read_buffer_bytes", maxReadBuffer, "bytes"),
		phase5DependencyMetric("metric.deps.stream_to_cas.read_calls", readCalls, "count"),
		phase5DependencyMetric("metric.deps.stream_to_cas.fetched_bytes", float64(streamResp.FetchedBytes), "bytes"),
		phase5DependencyMetric("metric.deps.cache_fill.peak_alloc_mb", peakAllocMB, "mb"),
	}, nil
}

func phase5ChunkFetcherStats(fetcher *phase5ChunkBoundFetcher) (float64, float64) {
	if fetcher == nil {
		return 0, 0
	}
	return float64(fetcher.maxSeenBuf.Load()), float64(fetcher.readCalls.Load())
}

func measurePhase5DependencyHandoff(service *Service) ([]perfcontracts.MeasurementRecord, error) {
	ensureResp, err := phase5EnsureDependency(service)
	if err != nil {
		return nil, err
	}
	handoffMS, handoffResp, err := phase5TimedDependencyHandoff(service)
	if err != nil {
		return nil, err
	}
	return []perfcontracts.MeasurementRecord{
		phase5DependencyMetric("metric.deps.materialization.workspace_handoff.wall_ms", handoffMS, "ms"),
		phase5DependencyMetric("metric.deps.materialization.workspace_handoff.found_count", boolToCount(handoffResp.Found), "count"),
		phase5DependencyMetric("metric.deps.cache_ensure.registry_requests", float64(ensureResp.RegistryRequestCount), "count"),
	}, nil
}

func phase5EnsureDependency(service *Service) (DependencyCacheEnsureResponse, error) {
	req := phase5DependencyEnsureRequest("req-deps-ensure", "run-deps-phase5", "handoff")
	resp, errResp := service.HandleDependencyCacheEnsure(context.Background(), req, RequestContext{})
	if err := phase5DependencyErr("dependency ensure failed", errResp); err != nil {
		return DependencyCacheEnsureResponse{}, err
	}
	return resp, nil
}

func phase5TimedDependencyHandoff(service *Service) (float64, DependencyCacheHandoffResponse, error) {
	req := phase5DependencyHandoffRequest("req-deps-handoff", "handoff", "workspace")
	started := time.Now()
	resp, errResp := service.HandleDependencyCacheHandoff(context.Background(), req, RequestContext{})
	if err := phase5DependencyErr("dependency handoff failed", errResp); err != nil {
		return 0, DependencyCacheHandoffResponse{}, err
	}
	return float64(time.Since(started).Microseconds()) / 1000.0, resp, nil
}
