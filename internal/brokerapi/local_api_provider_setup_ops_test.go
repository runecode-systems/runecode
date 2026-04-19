package brokerapi

import (
	"context"
	"reflect"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestProviderSetupSecretIngressFlowStoresOnlySecretRefAndIssuesModelGatewayLease(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	beginResp := mustBeginProviderSetup(t, service, "req-provider-setup-begin")
	assertDirectCredentialBeginProjection(t, beginResp)
	prepareResp := mustPrepareProviderIngress(t, service, beginResp.SetupSession.SetupSessionID, "req-provider-setup-prepare")
	submitResp := mustSubmitProviderIngress(t, service, prepareResp.SecretIngressToken, "super-secret-api-key", "req-provider-setup-submit")
	if got := submitResp.SetupSession.CurrentPhase; got != providerSetupPhaseCredentialCommitted {
		t.Fatalf("setup_session.current_phase = %q, want %s", got, providerSetupPhaseCredentialCommitted)
	}
	if got := submitResp.SetupSession.ValidationStatus; got != providerSetupValidationStatusNotStarted {
		t.Fatalf("setup_session.validation_status = %q, want %s", got, providerSetupValidationStatusNotStarted)
	}
	assertStoredAndProjectedCredentialState(t, service, beginResp.Profile.ProviderProfileID, submitResp)
	assertProviderCredentialLeaseIssue(t, service, beginResp.Profile.ProviderProfileID)
	assertProviderProfileInspectionProjection(t, service, beginResp.Profile.ProviderProfileID)
}

func TestProviderValidationLifecycleTransitionsAndReadinessCommit(t *testing.T) {
	service, profileID := seededProviderValidationService(t, "req-provider-validation-begin-setup", "req-provider-validation-prepare", "req-provider-validation-submit")
	validationBegin := mustBeginProviderValidation(t, service, profileID, "req-provider-validation-begin", "")
	assertValidationBeginInProgress(t, validationBegin)
	validationCommit := mustCommitProviderValidation(t, service, profileID, validationBegin.ValidationAttemptID, "req-provider-validation-commit", "reachable", "compatible")
	assertValidationCommitSucceeded(t, validationBegin.ValidationAttemptID, validationCommit)
}

func TestProviderValidationCommitRejectsStaleAttemptAndRequiresInProgressSession(t *testing.T) {
	service, profileID := seededProviderValidationService(t, "req-provider-validation-guard-setup", "req-provider-validation-guard-prepare", "req-provider-validation-guard-submit")
	assertValidationCommitRejected(t, service, profileID, "provider-validation-attempt-stale", "req-provider-validation-commit-without-begin")
	firstBegin := mustBeginProviderValidation(t, service, profileID, "req-provider-validation-guard-begin-first", "")
	secondBegin := mustBeginProviderValidation(t, service, profileID, "req-provider-validation-guard-begin-second", "provider-validation-attempt-current")
	assertDistinctValidationAttempts(t, firstBegin, secondBegin)
	assertValidationCommitRejected(t, service, profileID, firstBegin.ValidationAttemptID, "req-provider-validation-commit-stale")
}

func TestProviderValidationCommitFailureDoesNotMutateProfileReadiness(t *testing.T) {
	service, profileID := seededProviderValidationService(t, "req-provider-validation-consistency-setup", "req-provider-validation-consistency-prepare", "req-provider-validation-consistency-submit")
	validationBegin := mustBeginProviderValidation(t, service, profileID, "req-provider-validation-consistency-begin", "")
	before := mustProviderProfileByID(t, service, profileID, "before commit")
	service.providerSetup.setPersistFunc(func(session ProviderSetupSession) error {
		if session.CurrentPhase == providerSetupPhaseReadinessCommitted {
			return context.DeadlineExceeded
		}
		return nil
	})
	assertValidationCommitFails(t, service, profileID, validationBegin.ValidationAttemptID, "req-provider-validation-consistency-commit")
	after := mustProviderProfileByID(t, service, profileID, "after failed commit")
	assertReadinessUnchanged(t, before.ReadinessPosture, after.ReadinessPosture)
}

func TestProviderValidationAndReadinessAuditEventsEmitted(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	beginResp := mustBeginProviderSetup(t, service, "req-provider-audit-begin-setup")
	prepareResp := mustPrepareProviderIngress(t, service, beginResp.SetupSession.SetupSessionID, "req-provider-audit-prepare")
	mustSubmitProviderIngress(t, service, prepareResp.SecretIngressToken, "audit-secret", "req-provider-audit-submit")
	validationBegin, errResp := service.HandleProviderValidationBegin(context.Background(), ProviderValidationBeginRequest{
		SchemaID:          "runecode.protocol.v0.ProviderValidationBeginRequest",
		SchemaVersion:     "0.1.0",
		RequestID:         "req-provider-audit-validation-begin",
		ProviderProfileID: beginResp.Profile.ProviderProfileID,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProviderValidationBegin error response: %+v", errResp)
	}
	if _, errResp := service.HandleProviderValidationCommit(context.Background(), ProviderValidationCommitRequest{
		SchemaID:            "runecode.protocol.v0.ProviderValidationCommitRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-provider-audit-validation-commit",
		ProviderProfileID:   beginResp.Profile.ProviderProfileID,
		ValidationAttemptID: validationBegin.ValidationAttemptID,
		ConnectivityState:   "unreachable",
		CompatibilityState:  "incompatible",
		ReasonCodes:         []string{"dns_resolution_failed"},
	}, RequestContext{}); errResp != nil {
		t.Fatalf("HandleProviderValidationCommit error response: %+v", errResp)
	}

	events, err := service.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	assertAuditEventTypePresent(t, events, brokerAuditEventTypeProviderProfile)
	assertAuditEventTypePresent(t, events, brokerAuditEventTypeProviderCredential)
	assertAuditEventTypePresent(t, events, brokerAuditEventTypeProviderValidation)
	assertAuditEventTypePresent(t, events, brokerAuditEventTypeProviderReadiness)
}

func assertAuditEventTypePresent(t *testing.T, events []artifacts.AuditEvent, eventType string) {
	t.Helper()
	for _, event := range events {
		if event.Type == eventType {
			return
		}
	}
	t.Fatalf("missing audit event type %q", eventType)
}

func assertStoredAndProjectedCredentialState(t *testing.T, service *Service, profileID string, submitResp ProviderSetupSecretIngressSubmitResponse) {
	t.Helper()
	if submitResp.Profile.AuthMaterial.SecretRef != "" {
		t.Fatalf("auth_material.secret_ref = %q, want redacted projection", submitResp.Profile.AuthMaterial.SecretRef)
	}
	if got := submitResp.Profile.AuthMaterial.MaterialState; got != "present" {
		t.Fatalf("auth_material.material_state = %q, want present", got)
	}
	stored, ok := service.providerProfileByID(profileID)
	if !ok {
		t.Fatal("provider profile missing after secret ingress submit")
	}
	if stored.AuthMaterial.SecretRef == "" {
		t.Fatal("stored auth_material.secret_ref empty, want persisted internal secretsd reference")
	}
	if got := submitResp.Profile.ReadinessPosture.CredentialState; got != "present" {
		t.Fatalf("readiness_posture.credential_state = %q, want present", got)
	}
	if got := submitResp.Profile.CompatibilityPosture; got != "unverified" {
		t.Fatalf("compatibility_posture = %q, want unverified", got)
	}
}

func assertProviderCredentialLeaseIssue(t *testing.T, service *Service, profileID string) {
	t.Helper()
	leaseResp, errResp := service.HandleProviderCredentialLeaseIssue(context.Background(), ProviderCredentialLeaseIssueRequest{
		SchemaID:          "runecode.protocol.v0.ProviderCredentialLeaseIssueRequest",
		SchemaVersion:     "0.1.0",
		RequestID:         "req-provider-lease",
		ProviderProfileID: profileID,
		RunID:             "run-lease-test",
		TTLSeconds:        120,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProviderCredentialLeaseIssue error response: %+v", errResp)
	}
	if got := leaseResp.Lease.RoleKind; got != "model-gateway" {
		t.Fatalf("lease.role_kind = %q, want model-gateway", got)
	}
	if got := leaseResp.Lease.DeliveryKind; got != "model_gateway" {
		t.Fatalf("lease.delivery_kind = %q, want model_gateway", got)
	}
}

func assertProviderProfileInspectionProjection(t *testing.T, service *Service, profileID string) {
	t.Helper()
	listResp, errResp := service.HandleProviderProfileList(context.Background(), ProviderProfileListRequest{
		SchemaID:      "runecode.protocol.v0.ProviderProfileListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-provider-profile-list",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProviderProfileList error response: %+v", errResp)
	}
	if len(listResp.Profiles) != 1 {
		t.Fatalf("profiles count = %d, want 1", len(listResp.Profiles))
	}
	if got := listResp.Profiles[0].SupportedAuthModes; len(got) != 1 || got[0] != "direct_credential" {
		t.Fatalf("supported_auth_modes = %#v, want [direct_credential]", got)
	}
	if got := listResp.Profiles[0].AuthMaterial.SecretRef; got != "" {
		t.Fatalf("list auth_material.secret_ref = %q, want redacted", got)
	}
	getResp, errResp := service.HandleProviderProfileGet(context.Background(), ProviderProfileGetRequest{
		SchemaID:          "runecode.protocol.v0.ProviderProfileGetRequest",
		SchemaVersion:     "0.1.0",
		RequestID:         "req-provider-profile-get",
		ProviderProfileID: profileID,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProviderProfileGet error response: %+v", errResp)
	}
	if got := getResp.Profile.ProviderProfileID; got != profileID {
		t.Fatalf("profile.provider_profile_id = %q, want %q", got, profileID)
	}
	if got := getResp.Profile.AuthMaterial.SecretRef; got != "" {
		t.Fatalf("get auth_material.secret_ref = %q, want redacted", got)
	}
}

func seededProviderValidationService(t *testing.T, beginRequestID, prepareRequestID, submitRequestID string) (*Service, string) {
	t.Helper()
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	beginResp := mustBeginProviderSetup(t, service, beginRequestID)
	prepareResp := mustPrepareProviderIngress(t, service, beginResp.SetupSession.SetupSessionID, prepareRequestID)
	mustSubmitProviderIngress(t, service, prepareResp.SecretIngressToken, "validation-secret", submitRequestID)
	return service, beginResp.Profile.ProviderProfileID
}

func mustBeginProviderValidation(t *testing.T, service *Service, profileID, requestID, attemptID string) ProviderValidationBeginResponse {
	t.Helper()
	resp, errResp := service.HandleProviderValidationBegin(context.Background(), ProviderValidationBeginRequest{
		SchemaID:            "runecode.protocol.v0.ProviderValidationBeginRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           requestID,
		ProviderProfileID:   profileID,
		ValidationAttemptID: attemptID,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProviderValidationBegin error response: %+v", errResp)
	}
	return resp
}

func mustCommitProviderValidation(t *testing.T, service *Service, profileID, attemptID, requestID, connectivityState, compatibilityState string) ProviderValidationCommitResponse {
	t.Helper()
	resp, errResp := service.HandleProviderValidationCommit(context.Background(), ProviderValidationCommitRequest{
		SchemaID:            "runecode.protocol.v0.ProviderValidationCommitRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           requestID,
		ProviderProfileID:   profileID,
		ValidationAttemptID: attemptID,
		ConnectivityState:   connectivityState,
		CompatibilityState:  compatibilityState,
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProviderValidationCommit error response: %+v", errResp)
	}
	return resp
}

func assertValidationBeginInProgress(t *testing.T, validationBegin ProviderValidationBeginResponse) {
	t.Helper()
	if got := validationBegin.SetupSession.CurrentPhase; got != providerSetupPhaseValidationInProgress {
		t.Fatalf("begin setup_session.current_phase = %q, want %s", got, providerSetupPhaseValidationInProgress)
	}
	if got := validationBegin.SetupSession.ValidationStatus; got != providerSetupValidationStatusInProgress {
		t.Fatalf("begin setup_session.validation_status = %q, want %s", got, providerSetupValidationStatusInProgress)
	}
	if validationBegin.ValidationAttemptID == "" {
		t.Fatal("validation_attempt_id empty, want broker-owned validation attempt identity")
	}
	if got := validationBegin.Profile.ReadinessPosture.ValidationAttemptID; got != validationBegin.ValidationAttemptID {
		t.Fatalf("profile.readiness_posture.validation_attempt_id = %q, want %q", got, validationBegin.ValidationAttemptID)
	}
	if !stringSliceContains(validationBegin.Profile.ReadinessPosture.ReasonCodes, "validation_in_progress") {
		t.Fatalf("profile.readiness_posture.reason_codes = %#v, want validation_in_progress", validationBegin.Profile.ReadinessPosture.ReasonCodes)
	}
}

func assertValidationCommitSucceeded(t *testing.T, attemptID string, validationCommit ProviderValidationCommitResponse) {
	t.Helper()
	if got := validationCommit.ValidationOutcome; got != "succeeded" {
		t.Fatalf("validation_outcome = %q, want succeeded", got)
	}
	if got := validationCommit.SetupSession.CurrentPhase; got != providerSetupPhaseReadinessCommitted {
		t.Fatalf("commit setup_session.current_phase = %q, want %s", got, providerSetupPhaseReadinessCommitted)
	}
	if !validationCommit.SetupSession.ReadinessCommitted {
		t.Fatal("setup_session.readiness_committed = false, want true")
	}
	if got := validationCommit.Profile.ReadinessPosture.EffectiveReadiness; got != "ready" {
		t.Fatalf("readiness_posture.effective_readiness = %q, want ready", got)
	}
	if got := validationCommit.Profile.CompatibilityPosture; got != "compatible" {
		t.Fatalf("compatibility_posture = %q, want compatible", got)
	}
	if got := validationCommit.Profile.ReadinessPosture.ValidationAttemptID; got != attemptID {
		t.Fatalf("commit profile.readiness_posture.validation_attempt_id = %q, want %q", got, attemptID)
	}
}

func assertValidationCommitRejected(t *testing.T, service *Service, profileID, attemptID, requestID string) {
	t.Helper()
	if _, errResp := service.HandleProviderValidationCommit(context.Background(), ProviderValidationCommitRequest{
		SchemaID:            "runecode.protocol.v0.ProviderValidationCommitRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           requestID,
		ProviderProfileID:   profileID,
		ValidationAttemptID: attemptID,
		ConnectivityState:   "reachable",
		CompatibilityState:  "compatible",
	}, RequestContext{}); errResp == nil {
		t.Fatal("expected validation commit rejection")
	} else if got := errResp.Error.Code; got != "broker_validation_schema_invalid" {
		t.Fatalf("validation commit code = %q, want broker_validation_schema_invalid", got)
	}
}

func assertDistinctValidationAttempts(t *testing.T, firstBegin, secondBegin ProviderValidationBeginResponse) {
	t.Helper()
	if firstBegin.ValidationAttemptID == secondBegin.ValidationAttemptID {
		t.Fatal("expected distinct validation attempts between first and second begin")
	}
}

func mustProviderProfileByID(t *testing.T, service *Service, profileID, label string) ProviderProfile {
	t.Helper()
	profile, ok := service.providerProfileByID(profileID)
	if !ok {
		t.Fatalf("provider profile not found %s", label)
	}
	return profile
}

func assertValidationCommitFails(t *testing.T, service *Service, profileID, attemptID, requestID string) {
	t.Helper()
	if _, errResp := service.HandleProviderValidationCommit(context.Background(), ProviderValidationCommitRequest{
		SchemaID:            "runecode.protocol.v0.ProviderValidationCommitRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           requestID,
		ProviderProfileID:   profileID,
		ValidationAttemptID: attemptID,
		ConnectivityState:   "reachable",
		CompatibilityState:  "compatible",
	}, RequestContext{}); errResp == nil {
		t.Fatal("expected validation commit failure when session persistence fails")
	}
}

func assertReadinessUnchanged(t *testing.T, before, after ProviderReadinessPosture) {
	t.Helper()
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("readiness_posture mutated on failed commit:\nbefore=%#v\nafter=%#v", before, after)
	}
}

func mustBeginProviderSetup(t *testing.T, service *Service, requestID string) ProviderSetupSessionBeginResponse {
	t.Helper()
	beginResp, errResp := service.HandleProviderSetupSessionBegin(context.Background(), ProviderSetupSessionBeginRequest{
		SchemaID:            "runecode.protocol.v0.ProviderSetupSessionBeginRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           requestID,
		DisplayLabel:        "OpenAI Prod",
		ProviderFamily:      "openai_compatible",
		AdapterKind:         "chat_completions_v0",
		CanonicalHost:       "api.openai.com",
		CanonicalPathPrefix: "/v1",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProviderSetupSessionBegin error response: %+v", errResp)
	}
	return beginResp
}

func assertDirectCredentialBeginProjection(t *testing.T, beginResp ProviderSetupSessionBeginResponse) {
	t.Helper()
	if got := beginResp.SetupSession.SupportedAuthModes; len(got) != 1 || got[0] != "direct_credential" {
		t.Fatalf("setup_session.supported_auth_modes = %#v, want [direct_credential]", got)
	}
	if got := beginResp.SetupSession.CurrentAuthMode; got != "direct_credential" {
		t.Fatalf("setup_session.current_auth_mode = %q, want direct_credential", got)
	}
	if got := beginResp.Profile.SupportedAuthModes; len(got) != 1 || got[0] != "direct_credential" {
		t.Fatalf("profile.supported_auth_modes = %#v, want [direct_credential]", got)
	}
	if got := beginResp.Profile.CurrentAuthMode; got != "direct_credential" {
		t.Fatalf("profile.current_auth_mode = %q, want direct_credential", got)
	}
	if got := beginResp.Profile.ModelCatalogPosture.SelectionAuthority; got != "manual_allowlist_canonical" {
		t.Fatalf("profile.model_catalog_posture.selection_authority = %q, want manual_allowlist_canonical", got)
	}
}

func mustPrepareProviderIngress(t *testing.T, service *Service, sessionID, requestID string) ProviderSetupSecretIngressPrepareResponse {
	t.Helper()
	prepareResp, errResp := service.HandleProviderSetupSecretIngressPrepare(context.Background(), ProviderSetupSecretIngressPrepareRequest{
		SchemaID:        "runecode.protocol.v0.ProviderSetupSecretIngressPrepareRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       requestID,
		SetupSessionID:  sessionID,
		IngressChannel:  "cli_stdin",
		CredentialField: "api_key",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProviderSetupSecretIngressPrepare error response: %+v", errResp)
	}
	return prepareResp
}

func mustSubmitProviderIngress(t *testing.T, service *Service, token, secret, requestID string) ProviderSetupSecretIngressSubmitResponse {
	t.Helper()
	submitResp, errResp := service.HandleProviderSetupSecretIngressSubmit(context.Background(), ProviderSetupSecretIngressSubmitRequest{
		SchemaID:           "runecode.protocol.v0.ProviderSetupSecretIngressSubmitRequest",
		SchemaVersion:      "0.1.0",
		RequestID:          requestID,
		SecretIngressToken: token,
	}, []byte(secret), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProviderSetupSecretIngressSubmit error response: %+v", errResp)
	}
	return submitResp
}

func assertProviderIngressSubmitRejected(t *testing.T, service *Service, token, requestID string) {
	t.Helper()
	if _, errResp := service.HandleProviderSetupSecretIngressSubmit(context.Background(), ProviderSetupSecretIngressSubmitRequest{
		SchemaID:           "runecode.protocol.v0.ProviderSetupSecretIngressSubmitRequest",
		SchemaVersion:      "0.1.0",
		RequestID:          requestID,
		SecretIngressToken: token,
	}, []byte("stale-secret"), RequestContext{}); errResp == nil {
		t.Fatal("expected older secret ingress token rejection")
	}
}

func TestProviderSetupRejectsForbiddenIngressChannel(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	beginResp, errResp := service.HandleProviderSetupSessionBegin(context.Background(), ProviderSetupSessionBeginRequest{
		SchemaID:            "runecode.protocol.v0.ProviderSetupSessionBeginRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-provider-setup-begin",
		DisplayLabel:        "OpenAI Prod",
		ProviderFamily:      "openai_compatible",
		AdapterKind:         "chat_completions_v0",
		CanonicalHost:       "api.openai.com",
		CanonicalPathPrefix: "/v1",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProviderSetupSessionBegin error response: %+v", errResp)
	}
	_, errResp = service.HandleProviderSetupSecretIngressPrepare(context.Background(), ProviderSetupSecretIngressPrepareRequest{
		SchemaID:        "runecode.protocol.v0.ProviderSetupSecretIngressPrepareRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       "req-provider-setup-prepare",
		SetupSessionID:  beginResp.SetupSession.SetupSessionID,
		IngressChannel:  "environment_variable",
		CredentialField: "api_key",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("expected forbidden ingress channel rejection")
	}
}

func TestProviderSetupRejectsAdapterFamilyMismatch(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := service.HandleProviderSetupSessionBegin(context.Background(), ProviderSetupSessionBeginRequest{
		SchemaID:            "runecode.protocol.v0.ProviderSetupSessionBeginRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           "req-provider-setup-begin-mismatch",
		DisplayLabel:        "Anthropic Prod",
		ProviderFamily:      "anthropic_compatible",
		AdapterKind:         providerAdapterKindOpenAIChatCompletionsV0,
		CanonicalHost:       "api.anthropic.com",
		CanonicalPathPrefix: "/v1",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("expected adapter-family mismatch rejection")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
}

func TestProviderSetupPrepareInvalidatesOlderIngressTokens(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	beginResp := mustBeginProviderSetup(t, service, "req-provider-setup-begin-rotate")
	first := mustPrepareProviderIngress(t, service, beginResp.SetupSession.SetupSessionID, "req-provider-setup-prepare-first")
	second := mustPrepareProviderIngress(t, service, beginResp.SetupSession.SetupSessionID, "req-provider-setup-prepare-second")
	assertProviderIngressSubmitRejected(t, service, first.SecretIngressToken, "req-provider-setup-submit-old-token")
	mustSubmitProviderIngress(t, service, second.SecretIngressToken, "fresh-secret", "req-provider-setup-submit-new-token")
}

func TestProviderProfileInspectionRemainsAvailableWithoutSecretsService(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	_, err := service.providerSubstrate.upsertProfile(providerProfileFixture("OpenAI Prod", "openai_compatible", "api.openai.com", "/v1"))
	if err != nil {
		t.Fatalf("upsertProfile returned error: %v", err)
	}
	service.secretsSvc = nil
	if _, errResp := service.HandleProviderProfileList(context.Background(), ProviderProfileListRequest{SchemaID: "runecode.protocol.v0.ProviderProfileListRequest", SchemaVersion: "0.1.0", RequestID: "req-provider-list-read-only"}, RequestContext{}); errResp != nil {
		t.Fatalf("HandleProviderProfileList should remain available without secrets service: %+v", errResp)
	}
	if _, errResp := service.HandleProviderProfileGet(context.Background(), ProviderProfileGetRequest{SchemaID: "runecode.protocol.v0.ProviderProfileGetRequest", SchemaVersion: "0.1.0", RequestID: "req-provider-get-read-only", ProviderProfileID: stableProviderProfileID("openai_compatible", "api.openai.com/v1")}, RequestContext{}); errResp != nil {
		t.Fatalf("HandleProviderProfileGet should remain available without secrets service: %+v", errResp)
	}
}
