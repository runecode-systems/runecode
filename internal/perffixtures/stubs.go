package perffixtures

import "context"

type StubProviderBackend struct{}

type StubProviderRequest struct {
	Prompt string
}

type StubProviderResponse struct {
	Text                  string
	StatusCode            int
	ProviderLatencyMillis int
}

func (StubProviderBackend) Invoke(_ context.Context, _ StubProviderRequest) StubProviderResponse {
	return StubProviderResponse{Text: "stubbed provider response", StatusCode: 200, ProviderLatencyMillis: 7}
}

type StubSecretsBackend struct{}

func (StubSecretsBackend) IssueLease(runID string, providerID string) string {
	return "lease.stub." + runID + "." + providerID
}

type StubExternalAnchorTarget struct{}

func (StubExternalAnchorTarget) Prepare() string {
	return "prepared"
}

func (StubExternalAnchorTarget) ExecuteFastComplete() string {
	return "completed"
}

func (StubExternalAnchorTarget) ExecuteDeferred() string {
	return "deferred"
}

func (StubExternalAnchorTarget) AdmitReceipt() string {
	return "admitted"
}
