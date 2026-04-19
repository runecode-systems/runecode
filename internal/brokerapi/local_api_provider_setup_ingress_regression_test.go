package brokerapi

import (
	"context"
	"fmt"
	"testing"
	"time"

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

func TestProviderSetupSecretIngressSubmitRejectsExpiredTokenWithDistinctMessage(t *testing.T) {
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	service.providerSetup.setNowFunc(func() time.Time { return now })
	beginResp := mustBeginProviderSetup(t, service, "req-provider-setup-begin-expired-ingress")
	prepareResp := mustPrepareProviderIngress(t, service, beginResp.SetupSession.SetupSessionID, "req-provider-setup-prepare-expired-ingress")
	service.providerSetup.setNowFunc(func() time.Time { return now.Add(6 * time.Minute) })
	if _, errResp := service.HandleProviderSetupSecretIngressSubmit(context.Background(), ProviderSetupSecretIngressSubmitRequest{
		SchemaID:           "runecode.protocol.v0.ProviderSetupSecretIngressSubmitRequest",
		SchemaVersion:      "0.1.0",
		RequestID:          "req-provider-setup-submit-expired-ingress",
		SecretIngressToken: prepareResp.SecretIngressToken,
	}, []byte("secret-after-expiry"), RequestContext{}); errResp == nil {
		t.Fatal("expected expired secret ingress token rejection")
	} else if got := errResp.Error.Code; got != "broker_validation_schema_invalid" {
		t.Fatalf("expired token rejection code = %q, want broker_validation_schema_invalid", got)
	} else if got := errResp.Error.Message; got != "secret ingress token expired" {
		t.Fatalf("expired token rejection message = %q, want secret ingress token expired", got)
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
