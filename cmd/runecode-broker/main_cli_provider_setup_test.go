package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestProviderSetupDirectUsesTrustedSecretIngressWithoutArgsOrEnv(t *testing.T) {
	setBrokerServiceForTest(t)
	restore := installSecretStdin(t, "sk-test-secret\n")
	defer restore()
	requestedOps := make([]string, 0, 3)
	installProviderSetupDirectDispatchStub(t, &requestedOps)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"provider-setup-direct", "--provider-family", "openai_compatible", "--canonical-host", "api.openai.com"}, stdout, stderr); err != nil {
		t.Fatalf("provider-setup-direct returned error: %v", err)
	}
	if got := strings.Join(requestedOps, ","); got != "provider_setup_session_begin,provider_setup_secret_ingress_prepare,provider_setup_secret_ingress_submit" {
		t.Fatalf("requested ops = %q", got)
	}
}

func installSecretStdin(t *testing.T, input string) func() {
	t.Helper()
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe returned error: %v", err)
	}
	_, _ = stdinW.WriteString(input)
	_ = stdinW.Close()
	originalStdin := os.Stdin
	os.Stdin = stdinR
	return func() {
		os.Stdin = originalStdin
		_ = stdinR.Close()
	}
}

func installProviderSetupDirectDispatchStub(t *testing.T, requestedOps *[]string) {
	t.Helper()
	originalDispatch := localRPCDispatch
	localRPCDispatch = func(_ *brokerapi.Service, _ context.Context, wire localRPCRequest, _ brokerapi.RequestContext) localRPCResponse {
		*requestedOps = append(*requestedOps, wire.Operation)
		switch wire.Operation {
		case "provider_setup_session_begin":
			return providerSetupBeginStubResponse(t)
		case "provider_setup_secret_ingress_prepare":
			return providerSetupPrepareStubResponse(t)
		case "provider_setup_secret_ingress_submit":
			return providerSetupSubmitStubResponse(t, wire)
		default:
			return localRPCResponse{OK: false}
		}
	}
	t.Cleanup(func() { localRPCDispatch = originalDispatch })
}

func providerSetupBeginStubResponse(t *testing.T) localRPCResponse {
	t.Helper()
	return mustOKLocalRPCResponse(t, brokerapi.ProviderSetupSessionBeginResponse{SchemaID: "runecode.protocol.v0.ProviderSetupSessionBeginResponse", SchemaVersion: "0.1.0", RequestID: "req-begin", SetupSession: brokerapi.ProviderSetupSession{SchemaID: "runecode.protocol.v0.ProviderSetupSession", SchemaVersion: "0.1.0", SetupSessionID: "setup-1", ProviderProfileID: "profile-1", ProviderFamily: "openai_compatible", CurrentPhase: "metadata_configured", CurrentAuthMode: "direct_credential", SecretIngressReady: false, CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}, Profile: brokerapi.ProviderProfile{SchemaID: "runecode.protocol.v0.ProviderProfile", SchemaVersion: "0.1.0", ProviderProfileID: "profile-1", DisplayLabel: "OpenAI", ProviderFamily: "openai_compatible", AdapterKind: "chat_completions_v0", CurrentAuthMode: "direct_credential", SupportedAuthModes: []string{"direct_credential"}, AuthMaterial: brokerapi.ProviderAuthMaterial{SchemaID: "runecode.protocol.v0.ProviderAuthMaterial", SchemaVersion: "0.1.0", MaterialKind: "direct_credential", MaterialState: "missing"}, ReadinessPosture: brokerapi.ProviderReadinessPosture{SchemaID: "runecode.protocol.v0.ProviderReadinessPosture", SchemaVersion: "0.1.0", ConfigurationState: "configured", CredentialState: "missing", ConnectivityState: "unknown", CompatibilityState: "unknown", EffectiveReadiness: "not_ready"}}})
}

func providerSetupPrepareStubResponse(t *testing.T) localRPCResponse {
	t.Helper()
	return mustOKLocalRPCResponse(t, brokerapi.ProviderSetupSecretIngressPrepareResponse{SchemaID: "runecode.protocol.v0.ProviderSetupSecretIngressPrepareResponse", SchemaVersion: "0.1.0", RequestID: "req-prepare", SetupSession: brokerapi.ProviderSetupSession{SchemaID: "runecode.protocol.v0.ProviderSetupSession", SchemaVersion: "0.1.0", SetupSessionID: "setup-1", ProviderProfileID: "profile-1", ProviderFamily: "openai_compatible", CurrentPhase: "secret_ingress_ready", CurrentAuthMode: "direct_credential", SecretIngressReady: true, CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}, SecretIngressToken: "token-1", ExpiresAt: "2026-01-01T00:05:00Z"})
}

func providerSetupSubmitStubResponse(t *testing.T, wire localRPCRequest) localRPCResponse {
	t.Helper()
	decodedSecret, decodeErr := base64.StdEncoding.DecodeString(strings.TrimSpace(wire.SecretIngressPayloadBase64))
	if decodeErr != nil {
		t.Fatalf("DecodeString secret ingress payload returned error: %v", decodeErr)
	}
	if got := string(decodedSecret); got != "sk-test-secret" {
		t.Fatalf("secret payload = %q, want sk-test-secret", got)
	}
	req := brokerapi.ProviderSetupSecretIngressSubmitRequest{}
	if err := json.Unmarshal(wire.Request, &req); err != nil {
		t.Fatalf("Unmarshal submit request returned error: %v", err)
	}
	if req.SecretIngressToken != "token-1" {
		t.Fatalf("secret_ingress_token = %q, want token-1", req.SecretIngressToken)
	}
	return mustOKLocalRPCResponse(t, brokerapi.ProviderSetupSecretIngressSubmitResponse{SchemaID: "runecode.protocol.v0.ProviderSetupSecretIngressSubmitResponse", SchemaVersion: "0.1.0", RequestID: "req-submit", SetupSession: brokerapi.ProviderSetupSession{SchemaID: "runecode.protocol.v0.ProviderSetupSession", SchemaVersion: "0.1.0", SetupSessionID: "setup-1", ProviderProfileID: "profile-1", ProviderFamily: "openai_compatible", CurrentPhase: "configured", CurrentAuthMode: "direct_credential", SecretIngressReady: false, CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:01:00Z"}, Profile: brokerapi.ProviderProfile{SchemaID: "runecode.protocol.v0.ProviderProfile", SchemaVersion: "0.1.0", ProviderProfileID: "profile-1", DisplayLabel: "OpenAI", ProviderFamily: "openai_compatible", AdapterKind: "chat_completions_v0", CurrentAuthMode: "direct_credential", SupportedAuthModes: []string{"direct_credential"}, AuthMaterial: brokerapi.ProviderAuthMaterial{SchemaID: "runecode.protocol.v0.ProviderAuthMaterial", SchemaVersion: "0.1.0", MaterialKind: "direct_credential", MaterialState: "present", SecretRef: "secrets/model-providers/profile-1/direct-credential", LeasePolicyRef: "secretsd://lease-policy/model-provider-default"}, ReadinessPosture: brokerapi.ProviderReadinessPosture{SchemaID: "runecode.protocol.v0.ProviderReadinessPosture", SchemaVersion: "0.1.0", ConfigurationState: "configured", CredentialState: "present", ConnectivityState: "unknown", CompatibilityState: "unknown", EffectiveReadiness: "not_ready"}}})
}

func TestProviderSetupDirectRejectsEnvSecretInjection(t *testing.T) {
	setBrokerServiceForTest(t)
	t.Setenv("RUNE_PROVIDER_API_KEY", "do-not-use-env")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := run([]string{"provider-setup-direct", "--provider-family", "openai_compatible", "--canonical-host", "api.openai.com"}, stdout, stderr)
	if err == nil {
		t.Fatal("expected env injection rejection")
	}
	if !strings.Contains(err.Error(), "forbids secret environment-variable injection") {
		t.Fatalf("error = %v, want env-forbidden message", err)
	}
}

func TestProviderProfileInspectionCommandsUseBrokerOwnedRPCContracts(t *testing.T) {
	setBrokerServiceForTest(t)
	requestedOps := []string{}
	originalDispatch := localRPCDispatch
	localRPCDispatch = func(_ *brokerapi.Service, _ context.Context, wire localRPCRequest, _ brokerapi.RequestContext) localRPCResponse {
		requestedOps = append(requestedOps, wire.Operation)
		switch wire.Operation {
		case "provider_profile_list":
			return mustOKLocalRPCResponse(t, brokerapi.ProviderProfileListResponse{SchemaID: "runecode.protocol.v0.ProviderProfileListResponse", SchemaVersion: "0.1.0", RequestID: "req-list", Profiles: []brokerapi.ProviderProfile{}})
		case "provider_profile_get":
			return mustOKLocalRPCResponse(t, brokerapi.ProviderProfileGetResponse{SchemaID: "runecode.protocol.v0.ProviderProfileGetResponse", SchemaVersion: "0.1.0", RequestID: "req-get", Profile: brokerapi.ProviderProfile{SchemaID: "runecode.protocol.v0.ProviderProfile", SchemaVersion: "0.1.0", ProviderProfileID: "profile-1", DisplayLabel: "OpenAI", ProviderFamily: "openai_compatible", AdapterKind: "chat_completions_v0", DestinationRef: "model_endpoint://api.openai.com/v1", SupportedAuthModes: []string{"direct_credential"}, CurrentAuthMode: "direct_credential", AllowlistedModelIDs: []string{}, ModelCatalogPosture: brokerapi.ProviderModelCatalogPosture{SchemaID: "runecode.protocol.v0.ProviderModelCatalogPosture", SchemaVersion: "0.1.0", SelectionAuthority: "manual_allowlist_canonical", DiscoveryPosture: "advisory", CompatibilityProbePosture: "advisory"}, CompatibilityPosture: "unverified", QuotaProfileKind: "hybrid", RequestBindingKind: "canonical_llm_request_digest", SurfaceChannel: "broker_local_api", AuthMaterial: brokerapi.ProviderAuthMaterial{SchemaID: "runecode.protocol.v0.ProviderAuthMaterial", SchemaVersion: "0.1.0", MaterialKind: "direct_credential", MaterialState: "present"}, ReadinessPosture: brokerapi.ProviderReadinessPosture{SchemaID: "runecode.protocol.v0.ProviderReadinessPosture", SchemaVersion: "0.1.0", ConfigurationState: "configured", CredentialState: "present", ConnectivityState: "unknown", CompatibilityState: "unknown", EffectiveReadiness: "not_ready"}, Lifecycle: brokerapi.ProviderLifecycleMetadata{CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}}})
		default:
			return localRPCResponse{OK: false}
		}
	}
	t.Cleanup(func() { localRPCDispatch = originalDispatch })
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"provider-profile-list"}, stdout, stderr); err != nil {
		t.Fatalf("provider-profile-list returned error: %v", err)
	}
	stdout.Reset()
	if err := run([]string{"provider-profile-get", "--provider-profile-id", "profile-1"}, stdout, stderr); err != nil {
		t.Fatalf("provider-profile-get returned error: %v", err)
	}
	if got := strings.Join(requestedOps, ","); got != "provider_profile_list,provider_profile_get" {
		t.Fatalf("requested ops = %q", got)
	}
}
