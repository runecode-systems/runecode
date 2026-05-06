package brokerapi

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

func measurePhase5ExternalAnchorStubbed(trials int) []perfcontracts.MeasurementRecord {
	target := newPhase5ExternalAnchorStub()
	prepareSamples, deferredSamples, completedSamples, visibilitySamples, receiptSamples := phase5AnchorSamples(trials, target)
	prepareP95, _ := phase5P95(prepareSamples)
	deferredP95, _ := phase5P95(deferredSamples)
	completedP95, _ := phase5P95(completedSamples)
	visibilityP95, _ := phase5P95(visibilitySamples)
	receiptP95, _ := phase5P95(receiptSamples)
	return []perfcontracts.MeasurementRecord{
		{MetricID: "metric.anchor.prepare.latency.p95_ms", Value: prepareP95, Unit: "ms"},
		{MetricID: "metric.anchor.execute.completed.p95_ms", Value: completedP95, Unit: "ms"},
		{MetricID: "metric.anchor.execute.deferred.handoff.p95_ms", Value: deferredP95, Unit: "ms"},
		{MetricID: "metric.anchor.deferred.visibility.p95_ms", Value: visibilityP95, Unit: "ms"},
		{MetricID: "metric.anchor.receipt_admission.unchanged_seal.p95_ms", Value: receiptP95, Unit: "ms"},
		{MetricID: "metric.anchor.network_io_under_ledger_lock.count", Value: float64(target.networkUnderLock.Load()), Unit: "count"},
		{MetricID: "metric.anchor.verifier_bypass.count", Value: float64(target.verifierBypass.Load()), Unit: "count"},
	}
}

func phase5AnchorSamples(trials int, target *phase5ExternalAnchorStub) ([]float64, []float64, []float64, []float64, []float64) {
	prepareSamples := make([]float64, 0, trials)
	deferredSamples := make([]float64, 0, trials)
	completedSamples := make([]float64, 0, trials)
	visibilitySamples := make([]float64, 0, trials)
	receiptSamples := make([]float64, 0, trials)
	for i := 0; i < trials; i++ {
		prepareMS, completedMS, deferredMS, visibilityMS, receiptMS := phase5AnchorTrial(i, target)
		prepareSamples = append(prepareSamples, prepareMS)
		completedSamples = append(completedSamples, completedMS)
		deferredSamples = append(deferredSamples, deferredMS)
		visibilitySamples = append(visibilitySamples, visibilityMS)
		receiptSamples = append(receiptSamples, receiptMS)
	}
	return prepareSamples, deferredSamples, completedSamples, visibilitySamples, receiptSamples
}

func phase5AnchorTrial(i int, target *phase5ExternalAnchorStub) (float64, float64, float64, float64, float64) {
	seal := fmt.Sprintf("sha256:%064d", i+1)
	prepareMS := phase5MeasureMS(func() { target.Prepare(seal) })
	completedMS := phase5MeasureMS(func() { target.ExecuteFastComplete(seal) })
	deferredSeal := fmt.Sprintf("sha256:%064d", i+100)
	target.Prepare(deferredSeal)
	var requestID string
	deferredMS := phase5MeasureMS(func() { requestID = target.ExecuteDeferred(deferredSeal) })
	visibilityMS := phase5MeasureMS(func() { _ = target.WaitCompleted(requestID, 2*time.Second) })
	receiptMS := phase5MeasureMS(func() { target.AdmitReceiptUnchangedSeal(seal) })
	return prepareMS, completedMS, deferredMS, visibilityMS, receiptMS
}

type phase5ExternalAnchorStub struct {
	mu               sync.Mutex
	statusByRequest  map[string]string
	sealVerified     map[string]struct{}
	nextID           int
	networkUnderLock atomic.Int64
	verifierBypass   atomic.Int64
}

func newPhase5ExternalAnchorStub() *phase5ExternalAnchorStub {
	return &phase5ExternalAnchorStub{statusByRequest: map[string]string{}, sealVerified: map[string]struct{}{}}
}

func (s *phase5ExternalAnchorStub) Prepare(sealDigest string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sealVerified[sealDigest] = struct{}{}
}

func (s *phase5ExternalAnchorStub) ExecuteFastComplete(_ string) { time.Sleep(1 * time.Millisecond) }

func (s *phase5ExternalAnchorStub) ExecuteDeferred(_ string) string {
	s.mu.Lock()
	s.nextID++
	requestID := fmt.Sprintf("deferred-%d", s.nextID)
	s.statusByRequest[requestID] = "deferred"
	s.mu.Unlock()
	go func(id string) {
		time.Sleep(5 * time.Millisecond)
		s.mu.Lock()
		s.statusByRequest[id] = "completed"
		s.mu.Unlock()
	}(requestID)
	return requestID
}

func (s *phase5ExternalAnchorStub) WaitCompleted(requestID string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		s.mu.Lock()
		status := s.statusByRequest[requestID]
		s.mu.Unlock()
		if status == "completed" {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func (s *phase5ExternalAnchorStub) AdmitReceiptUnchangedSeal(sealDigest string) {
	s.mu.Lock()
	_, ok := s.sealVerified[sealDigest]
	s.mu.Unlock()
	if !ok {
		s.verifierBypass.Add(1)
	}
	time.Sleep(1 * time.Millisecond)
}
