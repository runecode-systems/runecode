package brokerapi

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDependencyFetchRegistryDigestMismatchRollsBackPayloadArtifact(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	fetcher := &mismatchDigestFetcher{payload: "payload-with-bad-expected-digest"}
	s.SetDependencyRegistryFetcherForTests(fetcher)

	before := s.List()
	_, errResp := s.HandleDependencyFetchRegistry(context.Background(), dependencyFetchRegistryRequestForTest("req-mismatch", "run-deps", "mismatch"), RequestContext{})
	if errResp == nil {
		t.Fatal("HandleDependencyFetchRegistry expected digest mismatch error")
	}
	after := s.List()
	if len(after) != len(before) {
		t.Fatalf("artifact count after mismatch = %d, want %d", len(after), len(before))
	}
}

func TestDependencyFetchRegistryCoalescesMisses(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{DependencyFetch: DependencyFetchConfig{MaxParallelFetches: 8}})
	fetcher := &gatedCountingFetcher{gate: make(chan struct{}), started: make(chan struct{})}
	s.SetDependencyRegistryFetcherForTests(fetcher)

	request := dependencyFetchRegistryRequestForTest("req-fetch-coalesce", "run-deps", "alpha")
	responses, errs := runCoalescedRegistryFetches(t, s, fetcher, request, 6)
	for i := range errs {
		if errs[i] != nil {
			t.Fatalf("caller %d error: %+v", i, errs[i])
		}
		if responses[i].RequestHash != request.RequestHash {
			t.Fatalf("caller %d request_hash mismatch", i)
		}
	}
	if got := fetcher.calls.Load(); got != 1 {
		t.Fatalf("fetcher calls = %d, want exact single-flight call", got)
	}
}

func runCoalescedRegistryFetches(t *testing.T, s *Service, fetcher *gatedCountingFetcher, request DependencyFetchRegistryRequest, callers int) ([]DependencyFetchRegistryResponse, []*ErrorResponse) {
	t.Helper()
	responses := make([]DependencyFetchRegistryResponse, callers)
	errs := make([]*ErrorResponse, callers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			fetcher.entered.Add(1)
			localReq := request
			localReq.RequestID = localReq.RequestID + "-" + string(rune('a'+idx))
			resp, errResp := s.HandleDependencyFetchRegistry(context.Background(), localReq, RequestContext{})
			responses[idx] = resp
			errs[idx] = errResp
		}(i)
	}
	close(start)
	waitForCoalescedFetchStart(t, fetcher, callers)
	close(fetcher.gate)
	wg.Wait()
	return responses, errs
}

func waitForCoalescedFetchStart(t *testing.T, fetcher *gatedCountingFetcher, callers int) {
	t.Helper()
	select {
	case <-fetcher.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first coalesced fetch start")
	}
	deadline := time.Now().Add(2 * time.Second)
	for fetcher.entered.Load() < int64(callers) {
		if time.Now().After(deadline) {
			t.Fatalf("entered callers = %d, want %d", fetcher.entered.Load(), callers)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestDependencyFetchRegistryBoundedParallelism(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{DependencyFetch: DependencyFetchConfig{MaxParallelFetches: 2}})
	fetcher := &concurrencyCountingFetcher{gate: make(chan struct{})}
	s.SetDependencyRegistryFetcherForTests(fetcher)

	requests := []DependencyFetchRegistryRequest{
		dependencyFetchRegistryRequestForTest("req-par-1", "run-deps", "a"),
		dependencyFetchRegistryRequestForTest("req-par-2", "run-deps", "b"),
		dependencyFetchRegistryRequestForTest("req-par-3", "run-deps", "c"),
		dependencyFetchRegistryRequestForTest("req-par-4", "run-deps", "d"),
	}

	var wg sync.WaitGroup
	for i := range requests {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, errResp := s.HandleDependencyFetchRegistry(context.Background(), requests[i], RequestContext{}); errResp != nil {
				t.Errorf("request %d error: %+v", i, errResp)
			}
		}(i)
	}
	close(fetcher.gate)
	wg.Wait()
	if got := fetcher.maxConcurrent.Load(); got > 2 {
		t.Fatalf("max concurrent fetches = %d, want <= 2", got)
	}
}

func TestDependencyFetchRegistryStreamsToCAS(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	payload := strings.Repeat("stream-me-", 8192)
	s.SetDependencyRegistryFetcherForTests(streamingFetcher{payload: payload})

	resp, errResp := s.HandleDependencyFetchRegistry(context.Background(), dependencyFetchRegistryRequestForTest("req-stream", "run-deps", "stream"), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleDependencyFetchRegistry error: %+v", errResp)
	}
	if len(resp.PayloadDigests) != 1 {
		t.Fatalf("payload_digests len = %d, want 1", len(resp.PayloadDigests))
	}
	payloadDigestIdentity, err := resp.PayloadDigests[0].Identity()
	if err != nil {
		t.Fatalf("payload digest identity error: %v", err)
	}
	r, err := s.Get(payloadDigestIdentity)
	if err != nil {
		t.Fatalf("Get payload digest returned error: %v", err)
	}
	b, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		t.Fatalf("ReadAll payload returned error: %v", readErr)
	}
	if string(b) != payload {
		t.Fatalf("stored payload mismatch")
	}
	if resp.FetchedBytes != int64(len(payload)) {
		t.Fatalf("fetched_bytes = %d, want %d", resp.FetchedBytes, len(payload))
	}
}

func TestDependencyFetchRegistryStreamsWithBoundedReadChunks(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	fetcher := &boundedChunkFetcher{payloadSize: 3 << 20, maxReadBuf: 128 << 10}
	s.SetDependencyRegistryFetcherForTests(fetcher)

	resp, errResp := s.HandleDependencyFetchRegistry(context.Background(), dependencyFetchRegistryRequestForTest("req-stream-bounded", "run-deps", "stream-bounded"), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleDependencyFetchRegistry error: %+v", errResp)
	}
	if resp.FetchedBytes != fetcher.payloadSize {
		t.Fatalf("fetched_bytes = %d, want %d", resp.FetchedBytes, fetcher.payloadSize)
	}
	if got := fetcher.maxSeenBuf.Load(); got > int64(fetcher.maxReadBuf) {
		t.Fatalf("max read buffer = %d, want <= %d", got, fetcher.maxReadBuf)
	}
	if got := fetcher.readCalls.Load(); got <= 1 {
		t.Fatalf("read calls = %d, want chunked streaming (>1)", got)
	}
}

func TestDependencyFetchIdentityPortableAcrossStoreRoots(t *testing.T) {
	left := newBrokerAPIServiceForTests(t, APIConfig{})
	right := newBrokerAPIServiceForTests(t, APIConfig{})
	leftReq := dependencyFetchRegistryRequestForTest("req-portable-left", "run-deps-portability-left", "portable")
	rightReq := dependencyFetchRegistryRequestForTest("req-portable-right", "run-deps-portability-right", "portable")

	leftResp, leftErr := left.HandleDependencyFetchRegistry(context.Background(), leftReq, RequestContext{})
	if leftErr != nil {
		t.Fatalf("left HandleDependencyFetchRegistry error: %+v", leftErr)
	}
	rightResp, rightErr := right.HandleDependencyFetchRegistry(context.Background(), rightReq, RequestContext{})
	if rightErr != nil {
		t.Fatalf("right HandleDependencyFetchRegistry error: %+v", rightErr)
	}
	if leftResp.RequestHash != rightResp.RequestHash {
		t.Fatalf("request_hash mismatch across store roots")
	}
	if leftResp.ResolvedUnitDigest != rightResp.ResolvedUnitDigest {
		t.Fatalf("resolved_unit_digest mismatch across store roots")
	}
}
