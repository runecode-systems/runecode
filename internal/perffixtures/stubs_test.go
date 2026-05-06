package perffixtures

import (
	"context"
	"testing"
)

func TestStubProviderSecretsAndExternalAnchorDeterministic(t *testing.T) {
	provider := StubProviderBackend{}
	resp := provider.Invoke(context.Background(), StubProviderRequest{Prompt: "hello"})
	if resp.StatusCode != 200 || resp.Text == "" || resp.ProviderLatencyMillis <= 0 {
		t.Fatalf("provider response = %#v, want deterministic non-empty success", resp)
	}

	secrets := StubSecretsBackend{}
	lease := secrets.IssueLease("run-1", "provider-1")
	if lease != "lease.stub.run-1.provider-1" {
		t.Fatalf("lease = %q, want deterministic lease id", lease)
	}

	anchor := StubExternalAnchorTarget{}
	if anchor.Prepare() != "prepared" || anchor.ExecuteFastComplete() != "completed" || anchor.ExecuteDeferred() != "deferred" || anchor.AdmitReceipt() != "admitted" {
		t.Fatal("external anchor stub returned unexpected state")
	}
}
