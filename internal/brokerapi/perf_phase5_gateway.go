package brokerapi

import (
	"context"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
	"github.com/runecode-ai/runecode/internal/perffixtures"
)

func measurePhase5GatewayAndSecrets(trials int) ([]perfcontracts.MeasurementRecord, error) {
	provider := perffixtures.StubProviderBackend{}
	secrets := perffixtures.StubSecretsBackend{}

	invokeSamples := make([]float64, 0, trials)
	leaseSamples := make([]float64, 0, trials)
	ingressSamples := make([]float64, 0, trials)
	for i := 0; i < trials; i++ {
		invokeSamples = append(invokeSamples, phase5MeasureMS(func() {
			_ = provider.Invoke(context.Background(), perffixtures.StubProviderRequest{Prompt: "phase5"})
		}))
		leaseSamples = append(leaseSamples, phase5MeasureMS(func() {
			_ = secrets.IssueLease("run-phase5", "provider-stub")
		}))
		ingressSamples = append(ingressSamples, phase5MeasureMS(func() {
			_ = secrets.IssueLease("run-phase5", "provider-stub")
		}))
	}
	invokeP95, err := phase5P95(invokeSamples)
	if err != nil {
		return nil, err
	}
	leaseP95, err := phase5P95(leaseSamples)
	if err != nil {
		return nil, err
	}
	ingressP95, err := phase5P95(ingressSamples)
	if err != nil {
		return nil, err
	}
	return []perfcontracts.MeasurementRecord{
		{MetricID: "metric.gateway.model_invoke.overhead.p95_ms", Value: invokeP95, Unit: "ms"},
		{MetricID: "metric.secrets.lease_issue.p95_ms", Value: leaseP95, Unit: "ms"},
		{MetricID: "metric.secrets.ingress.prepare_submit.p95_ms", Value: ingressP95, Unit: "ms"},
	}, nil
}

func phase5MeasureMS(call func()) float64 {
	started := time.Now()
	call()
	return float64(time.Since(started).Microseconds()) / 1000.0
}
