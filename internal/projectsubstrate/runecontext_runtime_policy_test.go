package projectsubstrate

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"
)

type stubRuntimePolicyProvider struct {
	policy runtimePolicySnapshot
	err    error
}

func (s stubRuntimePolicyProvider) RuntimePolicy() (runtimePolicySnapshot, error) {
	return s.policy, s.err
}

func TestReleaseCompatibilityPolicyUsesRuntimeMetadataPolicy(t *testing.T) {
	original := runtimeRuneContextPolicyProvider
	runtimeRuneContextPolicyProvider = stubRuntimePolicyProvider{policy: runtimePolicySnapshot{
		SupportedRuneContextVersionMin: "0.1.0-alpha.12",
		SupportedRuneContextVersionMax: "0.1.0-alpha.20",
		RecommendedRuneContextVersion:  "0.1.0-alpha.14",
		LocalRunectxVersion:            "0.1.0-alpha.14",
	}}
	t.Cleanup(func() { runtimeRuneContextPolicyProvider = original })

	policy := ReleaseCompatibilityPolicy()
	if got := policy.SupportedRuneContextVersionMin; got != "0.1.0-alpha.12" {
		t.Fatalf("supported_runecontext_version_min = %q, want 0.1.0-alpha.12", got)
	}
	if got := policy.SupportedRuneContextVersionMax; got != "0.1.0-alpha.20" {
		t.Fatalf("supported_runecontext_version_max = %q, want 0.1.0-alpha.20", got)
	}
	if got := policy.RecommendedRuneContextVersion; got != "0.1.0-alpha.14" {
		t.Fatalf("recommended_runecontext_version = %q, want 0.1.0-alpha.14", got)
	}
	if got := policy.DiagnosticsLocalRunectxVersion; got != "0.1.0-alpha.14" {
		t.Fatalf("diagnostics_local_runectx_version = %q, want 0.1.0-alpha.14", got)
	}
}

func TestSecureRunectxBinaryPathRejectsMissingBinary(t *testing.T) {
	original := resolveRunectxBinaryPath
	resolveRunectxBinaryPath = func() (string, error) {
		return "", errTestRuntimeUnavailable{}
	}
	t.Cleanup(func() { resolveRunectxBinaryPath = original })

	_, err := execRuntimePolicyProvider{}.RuntimePolicy()
	if err == nil {
		t.Fatal("RuntimePolicy error = nil, want resolution error")
	}
	if got := err.Error(); got == "" {
		t.Fatal("RuntimePolicy error empty, want resolution context")
	}
}

func TestReleaseCompatibilityPolicyFallsBackWhenRuntimeMetadataUnavailable(t *testing.T) {
	original := runtimeRuneContextPolicyProvider
	runtimeRuneContextPolicyProvider = stubRuntimePolicyProvider{err: errTestRuntimeUnavailable{}}
	t.Cleanup(func() { runtimeRuneContextPolicyProvider = original })

	policy := ReleaseCompatibilityPolicy()
	if got := policy.SupportedRuneContextVersionMin; got != releaseSupportedRuneContextVersionMin {
		t.Fatalf("supported_runecontext_version_min = %q, want %q", got, releaseSupportedRuneContextVersionMin)
	}
	if got := policy.SupportedRuneContextVersionMax; got != releaseSupportedRuneContextVersionMax {
		t.Fatalf("supported_runecontext_version_max = %q, want %q", got, releaseSupportedRuneContextVersionMax)
	}
	if got := policy.RecommendedRuneContextVersion; got != releaseRecommendedRuneContextVersion {
		t.Fatalf("recommended_runecontext_version = %q, want %q", got, releaseRecommendedRuneContextVersion)
	}
	if got := policy.DiagnosticsLocalRunectxVersion; got != "" {
		t.Fatalf("diagnostics_local_runectx_version = %q, want empty when falling back", got)
	}
}

func TestRuntimeCompatibilityPolicyFallbackLogUnavailableOmitsRawErrorDetail(t *testing.T) {
	originalProvider := runtimeRuneContextPolicyProvider
	runtimeRuneContextPolicyProvider = stubRuntimePolicyProvider{err: errSensitiveRuntimeFailure{}}
	t.Cleanup(func() { runtimeRuneContextPolicyProvider = originalProvider })

	logs := captureProjectSubstrateLogs(t, func() {
		_ = runtimeCompatibilityPolicy()
	})

	if !strings.Contains(logs, "runtime policy unavailable") {
		t.Fatalf("log output %q missing unavailable category", logs)
	}
	if strings.Contains(logs, "error=") {
		t.Fatalf("log output %q unexpectedly includes raw error field", logs)
	}
	if strings.Contains(logs, "secret/local/path") {
		t.Fatalf("log output %q leaked sensitive error detail", logs)
	}
}

func TestRuntimeCompatibilityPolicyFallbackLogInvalidOmitsRawErrorDetail(t *testing.T) {
	originalProvider := runtimeRuneContextPolicyProvider
	runtimeRuneContextPolicyProvider = stubRuntimePolicyProvider{policy: runtimePolicySnapshot{
		SupportedRuneContextVersionMin: "not-a-version",
		SupportedRuneContextVersionMax: "0.1.0-alpha.16",
		RecommendedRuneContextVersion:  "0.1.0-alpha.14",
	}}
	t.Cleanup(func() { runtimeRuneContextPolicyProvider = originalProvider })

	logs := captureProjectSubstrateLogs(t, func() {
		_ = runtimeCompatibilityPolicy()
	})

	if !strings.Contains(logs, "runtime policy invalid") {
		t.Fatalf("log output %q missing invalid category", logs)
	}
	if strings.Contains(logs, "error=") {
		t.Fatalf("log output %q unexpectedly includes raw error field", logs)
	}
	if strings.Contains(logs, "not-a-version") {
		t.Fatalf("log output %q leaked validation detail", logs)
	}
}

func TestDeriveRuntimePolicyFromMetadata(t *testing.T) {
	metadata := runeContextMetadataEnvelope{}
	metadata.Release.Version = "0.1.0-alpha.14"
	metadata.Compatibility.DefaultProjectVersion = "0.1.0-alpha.14"
	metadata.Compatibility.DirectlySupportedProjectVersion = []string{"0.1.0-alpha.5", "0.1.0-alpha.14"}
	metadata.Compatibility.UpgradeableFromProjectVersion = []string{"0.1.0-alpha.12", "0.1.0-alpha.13"}

	policy, err := deriveRuntimePolicy(metadata)
	if err != nil {
		t.Fatalf("deriveRuntimePolicy returned error: %v", err)
	}
	if got := policy.SupportedRuneContextVersionMin; got != "0.1.0-alpha.12" {
		t.Fatalf("supported min = %q, want 0.1.0-alpha.12", got)
	}
	if got := policy.SupportedRuneContextVersionMax; got != "0.1.0-alpha.14" {
		t.Fatalf("supported max = %q, want 0.1.0-alpha.14", got)
	}
	if got := policy.RecommendedRuneContextVersion; got != "0.1.0-alpha.14" {
		t.Fatalf("recommended = %q, want 0.1.0-alpha.14", got)
	}
	if got := policy.LocalRunectxVersion; got != "0.1.0-alpha.14" {
		t.Fatalf("local runectx version = %q, want 0.1.0-alpha.14", got)
	}
}

func TestDeriveRuntimePolicyWithoutUpgradeableVersionsUsesRecommendedAsMinimum(t *testing.T) {
	metadata := runeContextMetadataEnvelope{}
	metadata.Compatibility.DefaultProjectVersion = "0.1.0-alpha.14"
	metadata.Compatibility.DirectlySupportedProjectVersion = []string{"0.1.0-alpha.14"}

	policy, err := deriveRuntimePolicy(metadata)
	if err != nil {
		t.Fatalf("deriveRuntimePolicy returned error: %v", err)
	}
	if got := policy.SupportedRuneContextVersionMin; got != "0.1.0-alpha.14" {
		t.Fatalf("supported min = %q, want 0.1.0-alpha.14", got)
	}
}

type errTestRuntimeUnavailable struct{}

func (errTestRuntimeUnavailable) Error() string { return "runtime unavailable" }

type errSensitiveRuntimeFailure struct{}

func (errSensitiveRuntimeFailure) Error() string {
	return "exec: /secret/local/path/runectx: no such file or directory"
}

func captureProjectSubstrateLogs(t *testing.T, run func()) string {
	t.Helper()

	var buf bytes.Buffer
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
	})

	run()
	return strings.TrimSpace(fmt.Sprintf("%s", buf.String()))
}
