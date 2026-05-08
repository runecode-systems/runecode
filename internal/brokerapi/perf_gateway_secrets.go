package brokerapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func measurePhase5GatewayAndSecrets(trials int, repoRoot string) ([]perfcontracts.MeasurementRecord, error) {
	rig, err := newPhase5GatewayPerfRig(repoRoot)
	if err != nil {
		return nil, err
	}
	defer rig.cleanup()
	invokeP95, err := phase5TrialP95(trials, rig.invokeGatewayTrial)
	if err != nil {
		return nil, err
	}
	leaseP95, err := phase5TrialP95(trials, rig.issueLeaseTrial)
	if err != nil {
		return nil, err
	}
	ingressP95, err := phase5TrialP95(trials, rig.ingressPrepareSubmitTrial)
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

func phase5MeasureMSErr(call func() error) (float64, error) {
	started := time.Now()
	if err := call(); err != nil {
		return 0, err
	}
	return float64(time.Since(started).Microseconds()) / 1000.0, nil
}

func phase5TrialP95(trials int, trial func(int) error) (float64, error) {
	samples := make([]float64, 0, trials)
	for i := 0; i < trials; i++ {
		ms, err := phase5MeasureMSErr(func() error { return trial(i) })
		if err != nil {
			return 0, err
		}
		samples = append(samples, ms)
	}
	return phase5P95(samples)
}

type phase5GatewayPerfRig struct {
	service           *Service
	runID             string
	providerProfileID string
	llmRequest        map[string]any
	requestDigest     trustpolicy.Digest
	cleanupFn         func()
}

func newPhase5GatewayPerfRig(repoRoot string) (*phase5GatewayPerfRig, error) {
	service, cleanup, err := newPhase5GatewayPerfService(repoRoot)
	if err != nil {
		return nil, err
	}
	rig := &phase5GatewayPerfRig{service: service, runID: "run-phase5-gateway", cleanupFn: cleanup}
	if err := putPhase5TrustedModelGatewayContext(service, rig.runID); err != nil {
		rig.cleanup()
		return nil, err
	}
	if err := rig.seedProviderAndLLMRequest(); err != nil {
		rig.cleanup()
		return nil, err
	}
	return rig, nil
}

func (r *phase5GatewayPerfRig) cleanup() {
	if r != nil && r.cleanupFn != nil {
		r.cleanupFn()
	}
}

func (r *phase5GatewayPerfRig) invokeGatewayTrial(iteration int) error {
	req := LLMInvokeRequest{
		SchemaID:      "runecode.protocol.v0.LLMInvokeRequest",
		SchemaVersion: "0.1.0",
		RequestID:     fmt.Sprintf("req-phase5-llm-invoke-%d", iteration),
		RunID:         r.runID,
		LLMRequest:    r.llmRequest,
		RequestDigest: &r.requestDigest,
	}
	_, errResp := r.service.HandleLLMInvoke(context.Background(), req, RequestContext{})
	if errResp == nil {
		return nil
	}
	return fmt.Errorf("llm invoke: %s (%s)", errResp.Error.Code, strings.TrimSpace(errResp.Error.Message))
}

func (r *phase5GatewayPerfRig) issueLeaseTrial(iteration int) error {
	req := ProviderCredentialLeaseIssueRequest{
		SchemaID:          "runecode.protocol.v0.ProviderCredentialLeaseIssueRequest",
		SchemaVersion:     "0.1.0",
		RequestID:         fmt.Sprintf("req-phase5-lease-%d", iteration),
		ProviderProfileID: r.providerProfileID,
		RunID:             r.runID,
		TTLSeconds:        120,
	}
	_, errResp := r.service.HandleProviderCredentialLeaseIssue(context.Background(), req, RequestContext{})
	return phase5DependencyErr("provider lease issue", errResp)
}

func (r *phase5GatewayPerfRig) ingressPrepareSubmitTrial(iteration int) error {
	beginResp, err := r.beginProviderSetupSession(fmt.Sprintf("req-phase5-ingress-begin-%d", iteration), fmt.Sprintf("phase5-ingress-%d.example.com", iteration))
	if err != nil {
		return err
	}
	return r.submitIngressForSession(beginResp.SetupSession.SetupSessionID, fmt.Sprintf("%d", iteration))
}

func (r *phase5GatewayPerfRig) submitIngressForSession(setupSessionID string, suffix string) error {
	prepareResp, prepareErr := r.service.HandleProviderSetupSecretIngressPrepare(context.Background(), ProviderSetupSecretIngressPrepareRequest{
		SchemaID:        "runecode.protocol.v0.ProviderSetupSecretIngressPrepareRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       "req-phase5-ingress-prepare-" + suffix,
		SetupSessionID:  setupSessionID,
		IngressChannel:  "cli_stdin",
		CredentialField: "api_key",
	}, RequestContext{})
	if err := phase5DependencyErr("provider ingress prepare", prepareErr); err != nil {
		return err
	}
	_, submitErr := r.service.HandleProviderSetupSecretIngressSubmit(context.Background(), ProviderSetupSecretIngressSubmitRequest{
		SchemaID:           "runecode.protocol.v0.ProviderSetupSecretIngressSubmitRequest",
		SchemaVersion:      "0.1.0",
		RequestID:          "req-phase5-ingress-submit-" + suffix,
		SecretIngressToken: prepareResp.SecretIngressToken,
	}, []byte("phase5-secret"), RequestContext{})
	if err := phase5DependencyErr("provider ingress submit", submitErr); err != nil {
		return err
	}
	return nil
}
