package brokerapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/runecode-ai/runecode/internal/secretsd"
)

func TestProviderSetupSecretIngressTokenSpentBeforeCommitPreventsDuplicateImport(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	beginResp := mustBeginProviderSetup(t, service, "req-provider-setup-begin-atomic-ingress")
	prepareResp := mustPrepareProviderIngress(t, service, beginResp.SetupSession.SetupSessionID, "req-provider-setup-prepare-atomic-ingress")
	secretRef := fmt.Sprintf("secrets/model-providers/%s/direct-credential", beginResp.Profile.ProviderProfileID)
	service.providerSetup.setPersistFunc(func(session ProviderSetupSession) error {
		if session.CurrentPhase == providerSetupPhaseCredentialCommitted {
			return context.DeadlineExceeded
		}
		return nil
	})
	assertProviderIngressSubmitFailsForCommitPersistence(t, service, prepareResp.SecretIngressToken)
	firstMeta := mustLookupSecretMetadata(t, service, secretRef, "first submit attempt")

	service.providerSetup.setPersistFunc(nil)
	assertProviderIngressRetryRejected(t, service, prepareResp.SecretIngressToken)
	secondMeta := mustLookupSecretMetadata(t, service, secretRef, "retry rejection")
	if secondMeta.SecretID != firstMeta.SecretID {
		t.Fatalf("secret metadata changed on retry: first=%q second=%q", firstMeta.SecretID, secondMeta.SecretID)
	}
}

func assertProviderIngressSubmitFailsForCommitPersistence(t *testing.T, service *Service, token string) {
	t.Helper()
	if _, errResp := service.HandleProviderSetupSecretIngressSubmit(context.Background(), ProviderSetupSecretIngressSubmitRequest{
		SchemaID:           "runecode.protocol.v0.ProviderSetupSecretIngressSubmitRequest",
		SchemaVersion:      "0.1.0",
		RequestID:          "req-provider-setup-submit-atomic-ingress-fail",
		SecretIngressToken: token,
	}, []byte("first-secret"), RequestContext{}); errResp == nil {
		t.Fatal("expected secret ingress submit failure when credential-commit persistence fails")
	} else if got := errResp.Error.Code; got != "gateway_failure" {
		t.Fatalf("submit failure code = %q, want gateway_failure", got)
	}
}

func assertProviderIngressRetryRejected(t *testing.T, service *Service, token string) {
	t.Helper()
	if _, errResp := service.HandleProviderSetupSecretIngressSubmit(context.Background(), ProviderSetupSecretIngressSubmitRequest{
		SchemaID:           "runecode.protocol.v0.ProviderSetupSecretIngressSubmitRequest",
		SchemaVersion:      "0.1.0",
		RequestID:          "req-provider-setup-submit-atomic-ingress-retry",
		SecretIngressToken: token,
	}, []byte("second-secret"), RequestContext{}); errResp == nil {
		t.Fatal("expected spent secret ingress token rejection on retry")
	} else if got := errResp.Error.Code; got != "broker_validation_schema_invalid" {
		t.Fatalf("retry failure code = %q, want broker_validation_schema_invalid", got)
	}
}

func mustLookupSecretMetadata(t *testing.T, service *Service, secretRef, label string) secretsd.SecretMetadata {
	t.Helper()
	meta, ok := service.secretsSvc.LookupSecretMetadata(secretRef)
	if !ok {
		t.Fatalf("expected secret metadata after %s", label)
	}
	return meta
}
