package brokerapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProviderSetupSessionSurvivesRestartButIngressTokenIsInvalidated(t *testing.T) {
	root := canonicalTempDir(t)
	ledgerRoot := root + "/audit-ledger"
	secretsRoot := filepath.Join(root, "secretsd")
	t.Setenv("RUNE_SECRETS_STATE_ROOT", secretsRoot)

	first, err := NewServiceWithConfig(root, ledgerRoot, APIConfig{})
	if err != nil {
		t.Fatalf("NewServiceWithConfig(first) returned error: %v", err)
	}
	begin := mustBeginProviderSetup(t, first, "req-provider-durable-begin")
	prepare := mustPrepareProviderIngress(t, first, begin.SetupSession.SetupSessionID, "req-provider-durable-prepare")

	restarted, err := NewServiceWithConfig(root, ledgerRoot, APIConfig{})
	if err != nil {
		t.Fatalf("NewServiceWithConfig(restart) returned error: %v", err)
	}
	assertProviderIngressSubmitRejected(t, restarted, prepare.SecretIngressToken, "req-provider-durable-submit-stale")
	prepareAfterRestart := mustPrepareProviderIngress(t, restarted, begin.SetupSession.SetupSessionID, "req-provider-durable-prepare-restart")
	mustSubmitProviderIngress(t, restarted, prepareAfterRestart.SecretIngressToken, "restart-secret", "req-provider-durable-submit")
}

func TestProviderValidationLifecycleSessionPersistsAcrossRestart(t *testing.T) {
	root, ledgerRoot, _ := providerDurableRoots(t)
	first := mustNewProviderDurableService(t, root, ledgerRoot)
	profileID := mustSeedDurableProviderWithCredential(t, first, "req-provider-validation-durable-begin", "req-provider-validation-durable-prepare", "req-provider-validation-durable-submit", "durable-validation-secret")
	validationBegin := mustBeginProviderValidation(t, first, profileID, "req-provider-validation-durable-vbegin", "")
	restarted := mustNewProviderDurableService(t, root, ledgerRoot)
	validationCommit := mustCommitProviderValidation(t, restarted, profileID, validationBegin.ValidationAttemptID, "req-provider-validation-durable-vcommit", "reachable", "compatible")
	assertRestartValidationCommit(t, validationCommit)
}

func TestProviderProfileSurvivesRestartAndDegradesWhenSecretMissing(t *testing.T) {
	root, ledgerRoot, secretsRoot := providerDurableRoots(t)
	first := mustNewProviderDurableService(t, root, ledgerRoot)
	profileID := mustSeedDurableProviderWithCredential(t, first, "req-provider-profile-begin", "req-provider-profile-prepare", "req-provider-profile-submit", "first-secret")
	restarted := mustNewProviderDurableService(t, root, ledgerRoot)
	assertProviderRestartReady(t, restarted, profileID, "req-provider-profile-ready")
	mustRemoveSecretsRoot(t, secretsRoot)
	missing := mustNewProviderDurableService(t, root, ledgerRoot)
	assertProviderRestartMissing(t, missing, profileID, "req-provider-profile-missing")
}

func TestProviderProfileIncludedInBackupRestoreAndReconciledAgainstLocalSecrets(t *testing.T) {
	t.Setenv("RUNE_BACKUP_HMAC_KEY", "test-backup-key")
	sourceRoot, sourceLedger, _ := providerDurableRoots(t)
	source := mustNewProviderDurableService(t, sourceRoot, sourceLedger)
	profileID := mustSeedDurableProviderWithCredential(t, source, "req-provider-backup-begin", "req-provider-backup-prepare", "req-provider-backup-submit", "backup-secret")
	backupPath := filepath.Join(t.TempDir(), "provider-backup.json")
	mustExportBackup(t, source, backupPath)
	restoreRoot, restoreLedger, _ := providerDurableRoots(t)
	restored := mustNewProviderDurableService(t, restoreRoot, restoreLedger)
	mustRestoreBackup(t, restored, backupPath)
	assertProviderRestoreMissing(t, restored, profileID, "req-provider-backup-get")
}

func providerDurableRoots(t *testing.T) (string, string, string) {
	t.Helper()
	root := canonicalTempDir(t)
	ledgerRoot := root + "/audit-ledger"
	secretsRoot := filepath.Join(root, "secretsd")
	t.Setenv("RUNE_SECRETS_STATE_ROOT", secretsRoot)
	return root, ledgerRoot, secretsRoot
}

func mustNewProviderDurableService(t *testing.T, root, ledgerRoot string) *Service {
	t.Helper()
	service, err := NewServiceWithConfig(root, ledgerRoot, APIConfig{})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	return service
}

func mustSeedDurableProviderWithCredential(t *testing.T, service *Service, beginRequestID, prepareRequestID, submitRequestID, secret string) string {
	t.Helper()
	begin := mustBeginProviderSetup(t, service, beginRequestID)
	prepare := mustPrepareProviderIngress(t, service, begin.SetupSession.SetupSessionID, prepareRequestID)
	mustSubmitProviderIngress(t, service, prepare.SecretIngressToken, secret, submitRequestID)
	return begin.Profile.ProviderProfileID
}

func assertRestartValidationCommit(t *testing.T, validationCommit ProviderValidationCommitResponse) {
	t.Helper()
	if got := validationCommit.SetupSession.CurrentPhase; got != providerSetupPhaseReadinessCommitted {
		t.Fatalf("setup_session.current_phase after restart commit = %q, want %s", got, providerSetupPhaseReadinessCommitted)
	}
	if !validationCommit.SetupSession.ReadinessCommitted {
		t.Fatal("setup_session.readiness_committed after restart commit = false, want true")
	}
}

func assertProviderRestartReady(t *testing.T, service *Service, profileID, requestID string) {
	t.Helper()
	resp := mustGetProviderProfile(t, service, profileID, requestID)
	if got := resp.Profile.ReadinessPosture.CredentialState; got != "present" {
		t.Fatalf("credential_state after restart = %q, want present", got)
	}
}

func mustRemoveSecretsRoot(t *testing.T, secretsRoot string) {
	t.Helper()
	if err := os.RemoveAll(secretsRoot); err != nil {
		t.Fatalf("RemoveAll(secretsRoot) returned error: %v", err)
	}
}

func assertProviderRestartMissing(t *testing.T, service *Service, profileID, requestID string) {
	t.Helper()
	resp := mustGetProviderProfile(t, service, profileID, requestID)
	assertMissingProviderReadiness(t, resp, profileID)
}

func mustExportBackup(t *testing.T, service *Service, backupPath string) {
	t.Helper()
	if err := service.ExportBackup(backupPath); err != nil {
		t.Fatalf("ExportBackup returned error: %v", err)
	}
}

func mustRestoreBackup(t *testing.T, service *Service, backupPath string) {
	t.Helper()
	if err := service.RestoreBackup(backupPath); err != nil {
		t.Fatalf("RestoreBackup returned error: %v", err)
	}
}

func assertProviderRestoreMissing(t *testing.T, service *Service, profileID, requestID string) {
	t.Helper()
	resp := mustGetProviderProfile(t, service, profileID, requestID)
	assertMissingProviderReadiness(t, resp, profileID)
}

func mustGetProviderProfile(t *testing.T, service *Service, profileID, requestID string) ProviderProfileGetResponse {
	t.Helper()
	resp, errResp := service.HandleProviderProfileGet(context.Background(), ProviderProfileGetRequest{
		SchemaID:          "runecode.protocol.v0.ProviderProfileGetRequest",
		SchemaVersion:     "0.1.0",
		RequestID:         requestID,
		ProviderProfileID: profileID,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProviderProfileGet error response: %+v", errResp)
	}
	return resp
}

func assertMissingProviderReadiness(t *testing.T, resp ProviderProfileGetResponse, profileID string) {
	t.Helper()
	if got := resp.Profile.ProviderProfileID; got != profileID {
		t.Fatalf("provider_profile_id = %q, want %q", got, profileID)
	}
	if got := resp.Profile.AuthMaterial.MaterialState; got != "missing" {
		t.Fatalf("auth_material.material_state = %q, want missing", got)
	}
	if got := resp.Profile.ReadinessPosture.CredentialState; got != "missing" {
		t.Fatalf("readiness_posture.credential_state = %q, want missing", got)
	}
	if got := resp.Profile.ReadinessPosture.EffectiveReadiness; got != "not_ready" {
		t.Fatalf("readiness_posture.effective_readiness = %q, want not_ready", got)
	}
}
