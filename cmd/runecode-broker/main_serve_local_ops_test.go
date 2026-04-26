package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func TestProviderSetupRPCOperationsIncludeSecretSubmitAndLeaseIssue(t *testing.T) {
	ops := providerSetupRPCOperations(nil, context.Background(), brokerapi.RequestContext{})
	if _, ok := ops["provider_setup_secret_ingress_submit"]; !ok {
		t.Fatal("provider_setup_secret_ingress_submit operation missing")
	}
	if _, ok := ops["provider_credential_lease_issue"]; !ok {
		t.Fatal("provider_credential_lease_issue operation missing")
	}
	for _, op := range []string{"project_substrate_posture_get", "project_substrate_upgrade_preview", "project_substrate_upgrade_apply"} {
		if _, ok := ops[op]; !ok {
			t.Fatalf("%s operation missing", op)
		}
	}
}

func TestArtifactRPCOperationsIncludeDependencyCacheHandoff(t *testing.T) {
	ops := artifactRPCOperations(nil, context.Background(), brokerapi.RequestContext{})
	if _, ok := ops["dependency_cache_handoff"]; !ok {
		t.Fatal("dependency_cache_handoff operation missing")
	}
}

func TestDispatchLocalRPCProviderSetupSecretIngressSubmitPreservesPayloadAndRoutesLeaseIssue(t *testing.T) {
	service := newBrokerServiceWithSecretsState(t)
	ctx := context.Background()
	meta := brokerapi.RequestContext{ClientID: "test", LaneID: "test"}
	beginResp, errResp := beginProviderSetupSessionForTest(service, ctx, meta)
	if errResp != nil {
		t.Fatalf("HandleProviderSetupSessionBegin error response: %+v", errResp)
	}
	prepareResp, errResp := prepareProviderSetupSecretIngressForTest(service, ctx, beginResp.SetupSession.SetupSessionID, meta)
	if errResp != nil {
		t.Fatalf("HandleProviderSetupSecretIngressPrepare error response: %+v", errResp)
	}
	submitReqRaw := mustMarshalJSON(t, brokerapi.ProviderSetupSecretIngressSubmitRequest{
		SchemaID:           "runecode.protocol.v0.ProviderSetupSecretIngressSubmitRequest",
		SchemaVersion:      "0.1.0",
		RequestID:          "req-submit",
		SecretIngressToken: prepareResp.SecretIngressToken,
	})
	submitResp := localRPCDispatch(service, ctx, localRPCRequest{
		Operation:                  "provider_setup_secret_ingress_submit",
		Request:                    submitReqRaw,
		SecretIngressPayloadBase64: base64.StdEncoding.EncodeToString([]byte("super-secret-key")),
	}, meta)
	if !submitResp.OK || submitResp.Error != nil {
		t.Fatalf("provider_setup_secret_ingress_submit failed: %+v", submitResp.Error)
	}
	submitOut := brokerapi.ProviderSetupSecretIngressSubmitResponse{}
	if err := json.Unmarshal(submitResp.Response, &submitOut); err != nil {
		t.Fatalf("Unmarshal submit response error: %v", err)
	}
	if got := submitOut.Profile.AuthMaterial.MaterialState; got != "present" {
		t.Fatalf("submit profile auth_material.material_state = %q, want present", got)
	}
	leaseResp := localRPCDispatch(service, ctx, localRPCRequest{Operation: "provider_credential_lease_issue", Request: mustMarshalJSON(t, brokerapi.ProviderCredentialLeaseIssueRequest{
		SchemaID:          "runecode.protocol.v0.ProviderCredentialLeaseIssueRequest",
		SchemaVersion:     "0.1.0",
		RequestID:         "req-lease",
		ProviderProfileID: beginResp.Profile.ProviderProfileID,
		RunID:             "run-1",
		TTLSeconds:        120,
	})}, meta)
	if !leaseResp.OK || leaseResp.Error != nil {
		t.Fatalf("provider_credential_lease_issue failed: %+v", leaseResp.Error)
	}
	out := brokerapi.ProviderCredentialLeaseIssueResponse{}
	if err := json.Unmarshal(leaseResp.Response, &out); err != nil {
		t.Fatalf("Unmarshal lease response error: %v", err)
	}
	if got := out.Lease.RoleKind; got != "model-gateway" {
		t.Fatalf("lease.role_kind = %q, want model-gateway", got)
	}
}

func newBrokerServiceWithSecretsState(t *testing.T) *brokerapi.Service {
	t.Helper()
	root := filepath.Join(canonicalTempDir(t), "store")
	secretsRoot := filepath.Join(root, "secrets-state")
	seedBrokerSecretsReadinessState(t, secretsRoot)
	t.Setenv("RUNE_SECRETS_STATE_ROOT", secretsRoot)
	service, err := brokerapi.NewService(root, filepath.Join(root, "audit-ledger"))
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	return service
}

func beginProviderSetupSessionForTest(service *brokerapi.Service, ctx context.Context, meta brokerapi.RequestContext) (brokerapi.ProviderSetupSessionBeginResponse, *brokerapi.ErrorResponse) {
	return service.HandleProviderSetupSessionBegin(ctx, brokerapi.ProviderSetupSessionBeginRequest{
		SchemaID:            "runecode.protocol.v0.ProviderSetupSessionBeginRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-begin",
		DisplayLabel:        "OpenAI",
		ProviderFamily:      "openai_compatible",
		AdapterKind:         "chat_completions_v0",
		CanonicalHost:       "api.openai.com",
		CanonicalPathPrefix: "/v1",
	}, meta)
}

func prepareProviderSetupSecretIngressForTest(service *brokerapi.Service, ctx context.Context, setupSessionID string, meta brokerapi.RequestContext) (brokerapi.ProviderSetupSecretIngressPrepareResponse, *brokerapi.ErrorResponse) {
	return service.HandleProviderSetupSecretIngressPrepare(ctx, brokerapi.ProviderSetupSecretIngressPrepareRequest{
		SchemaID:        "runecode.protocol.v0.ProviderSetupSecretIngressPrepareRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       "req-prepare",
		SetupSessionID:  setupSessionID,
		IngressChannel:  "cli_stdin",
		CredentialField: "api_key",
	}, meta)
}

func mustMarshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	return b
}
