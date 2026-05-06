package brokerapi

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

func measurePhase5AuditVerification(
	repoRoot string,
	timeout time.Duration,
	runner func(repoRoot string, timeout time.Duration, command ...string) (float64, error),
) ([]perfcontracts.MeasurementRecord, error) {
	verifyMS, err := runner(repoRoot, timeout, "go", "test", "./internal/auditd", "-run", "TestVerifyCurrentSegmentIncrementalWithPreverifiedSealPersistsReport", "-count=1")
	if err != nil {
		return nil, fmt.Errorf("audit verify fixture check failed: %w", err)
	}
	finalizeMS, err := runner(repoRoot, timeout, "go", "test", "./internal/brokerapi", "-run", "TestHandleAuditFinalizeVerifyPersistsVerificationReportForCurrentSeal", "-count=1")
	if err != nil {
		return nil, fmt.Errorf("audit finalize verify fixture check failed: %w", err)
	}
	return []perfcontracts.MeasurementRecord{
		{MetricID: "metric.audit.verify_current_segment.wall_ms", Value: verifyMS, Unit: "ms"},
		{MetricID: "metric.audit.finalize_verify.wall_ms", Value: finalizeMS, Unit: "ms"},
	}, nil
}

func measurePhase5ProtocolChecks(
	repoRoot string,
	timeout time.Duration,
	runner func(repoRoot string, timeout time.Duration, command ...string) (float64, error),
) ([]perfcontracts.MeasurementRecord, error) {
	schemaMS, err := runner(repoRoot, timeout, "go", "test", "./internal/protocolschema")
	if err != nil {
		return nil, fmt.Errorf("protocol schema validation check failed: %w", err)
	}
	fixtureMS, err := runner(repoRoot, timeout, "node", "--test", "scripts/protocol-fixtures.test.js")
	if err != nil {
		return nil, fmt.Errorf("protocol fixture parity check failed: %w", err)
	}
	return []perfcontracts.MeasurementRecord{
		{MetricID: "metric.protocol.schema_validation.wall_ms", Value: schemaMS, Unit: "ms"},
		{MetricID: "metric.protocol.fixture_parity.wall_ms", Value: fixtureMS, Unit: "ms"},
	}, nil
}

func phase5RunCommand(repoRoot string, timeout time.Duration, command ...string) (float64, error) {
	if len(command) == 0 {
		return 0, fmt.Errorf("command required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = phase5CommandDir(repoRoot, command[0])
	start := time.Now()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, phase5CommandError(command, output, err)
	}
	return float64(time.Since(start).Microseconds()) / 1000.0, nil
}

func phase5CommandDir(repoRoot, bin string) string {
	if bin == "node" {
		return filepath.Join(repoRoot, "runner")
	}
	return repoRoot
}

func phase5CommandError(command []string, output []byte, runErr error) error {
	msg := strings.TrimSpace(string(output))
	if msg == "" {
		msg = runErr.Error()
	}
	return fmt.Errorf("%s failed: %s", strings.Join(command, " "), msg)
}

func phase5P95(values []float64) (float64, error) {
	if len(values) == 0 {
		return 0, fmt.Errorf("samples required")
	}
	cp := append([]float64(nil), values...)
	sort.Float64s(cp)
	idx := int(float64(len(cp)-1) * 0.95)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx], nil
}

func boolToCount(v bool) float64 {
	if v {
		return 1
	}
	return 0
}
