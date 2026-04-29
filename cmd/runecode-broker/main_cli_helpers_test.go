package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func mustJSONRawMessage(t *testing.T, value any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal JSON raw message error: %v", err)
	}
	return json.RawMessage(b)
}

func writeTempFile(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write temp file error: %v", err)
	}
	return path
}

func putArtifactViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, path, dataClass, provenance string, stateRoots ...string) artifacts.ArtifactReference {
	t.Helper()
	root := ""
	if len(stateRoots) > 0 {
		root = stateRoots[0]
	}
	if root == "" {
		root = setBrokerServiceForTest(t)
	}
	stdout.Reset()
	err := run([]string{"--state-root", root, "put-artifact", "--file", path, "--content-type", "text/plain", "--data-class", dataClass, "--provenance-hash", provenance}, stdout, stderr)
	if err != nil {
		t.Fatalf("put-artifact returned error: %v", err)
	}
	ref := artifacts.ArtifactReference{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &ref); unmarshalErr != nil {
		t.Fatalf("put-artifact output parse error: %v", unmarshalErr)
	}
	return ref
}

func listArtifactsViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, stateRoots ...string) []artifacts.ArtifactReference {
	t.Helper()
	stdout.Reset()
	args := []string{"list-artifacts"}
	if len(stateRoots) > 0 && stateRoots[0] != "" {
		args = []string{"--state-root", stateRoots[0], "list-artifacts"}
	}
	err := run(args, stdout, stderr)
	if err != nil {
		t.Fatalf("list-artifacts returned error: %v", err)
	}
	list := []artifacts.ArtifactReference{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &list); unmarshalErr != nil {
		t.Fatalf("list-artifacts output parse error: %v", unmarshalErr)
	}
	return list
}

func headArtifactViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, digest string, stateRoots ...string) artifacts.ArtifactReference {
	t.Helper()
	stdout.Reset()
	args := []string{"head-artifact", "--digest", digest}
	if len(stateRoots) > 0 && stateRoots[0] != "" {
		args = []string{"--state-root", stateRoots[0], "head-artifact", "--digest", digest}
	}
	err := run(args, stdout, stderr)
	if err != nil {
		t.Fatalf("head-artifact returned error: %v", err)
	}
	ref := artifacts.ArtifactReference{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &ref); unmarshalErr != nil {
		t.Fatalf("head-artifact output parse error: %v", unmarshalErr)
	}
	return ref
}

func getArtifactViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, digest, producer, consumer, dataClass string, manifestOptIn bool, out string, stateRoots ...string) {
	t.Helper()
	stdout.Reset()
	args := []string{"get-artifact", "--digest", digest, "--producer", producer, "--consumer", consumer, "--out", out}
	if len(stateRoots) > 0 && stateRoots[0] != "" {
		args = []string{"--state-root", stateRoots[0], "get-artifact", "--digest", digest, "--producer", producer, "--consumer", consumer, "--out", out}
	}
	if dataClass != "" {
		args = append(args, "--data-class", dataClass)
	}
	if manifestOptIn {
		args = append(args, "--manifest-opt-in")
	}
	err := run(args, stdout, stderr)
	if err != nil {
		t.Fatalf("get-artifact returned error: %v", err)
	}
}

func promoteViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, digest string, approvalRequestPath string, approvalEnvelopePath string) artifacts.ArtifactReference {
	t.Helper()
	seedPendingPromotionApprovalForCLI(t, digest, approvalRequestPath)
	stdout.Reset()
	err := run([]string{"promote-excerpt", "--unapproved-digest", digest, "--approver", "human", "--approval-request", approvalRequestPath, "--approval-envelope", approvalEnvelopePath, "--repo-path", "repo/file.txt", "--commit", "abc123", "--extractor-version", "tool-v1", "--full-content-visible"}, stdout, stderr)
	if err != nil {
		t.Fatalf("promote-excerpt returned error: %v", err)
	}
	approved := artifacts.ArtifactReference{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &approved); unmarshalErr != nil {
		t.Fatalf("approved parse error: %v", unmarshalErr)
	}
	return approved
}

func seedPendingPromotionApprovalForCLI(t *testing.T, digest, approvalRequestPath string) {
	t.Helper()
	service, err := brokerServiceFactory(defaultBrokerServiceRoots())
	if err != nil {
		t.Fatalf("brokerServiceFactory returned error: %v", err)
	}
	requestEnv, err := loadSignedApprovalEnvelope(approvalRequestPath)
	if err != nil {
		t.Fatalf("loadSignedApprovalEnvelope(%q) error: %v", approvalRequestPath, err)
	}
	approvalID, err := approvalRequestDigestForCLITests(*requestEnv)
	if err != nil {
		t.Fatalf("approvalRequestDigestForCLITests returned error: %v", err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(requestEnv.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal request payload error: %v", err)
	}
	seedPolicyDecisionForCLIApprovalRecord(t, service, payload)
	expiresAt := mustTimeForCLITests(t, stringField(payload, "expires_at"))
	if err := service.RecordApproval(artifacts.ApprovalRecord{
		ApprovalID:             approvalID,
		Status:                 "pending",
		WorkspaceID:            "",
		RunID:                  "",
		StageID:                "",
		ActionKind:             "promotion",
		RequestedAt:            mustTimeForCLITests(t, stringField(payload, "requested_at")),
		ExpiresAt:              &expiresAt,
		ApprovalTriggerCode:    stringField(payload, "approval_trigger_code"),
		ChangesIfApproved:      stringField(payload, "changes_if_approved"),
		ApprovalAssuranceLevel: stringField(payload, "approval_assurance_level"),
		PresenceMode:           stringField(payload, "presence_mode"),
		ManifestHash:           digestField(payload, "manifest_hash"),
		ActionRequestHash:      digestField(payload, "action_request_hash"),
		RelevantArtifactHashes: digestListField(payload, "relevant_artifact_hashes"),
		RequestDigest:          approvalID,
		SourceDigest:           digest,
		RequestEnvelope:        requestEnv,
	}); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
}

func seedPolicyDecisionForCLIApprovalRecord(t *testing.T, service *brokerapi.Service, payload map[string]any) {
	t.Helper()
	manifestHash := digestField(payload, "manifest_hash")
	actionHash := digestField(payload, "action_request_hash")
	if manifestHash == "" || actionHash == "" {
		t.Fatalf("approval payload missing manifest/action request hash for policy decision seeding")
	}
	decision := policyengine.PolicyDecision{
		SchemaID:               "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:          "0.3.0",
		DecisionOutcome:        policyengine.DecisionDeny,
		PolicyReasonCode:       "deny_by_default",
		ManifestHash:           manifestHash,
		ActionRequestHash:      actionHash,
		PolicyInputHashes:      []string{manifestHash},
		RelevantArtifactHashes: digestListField(payload, "relevant_artifact_hashes"),
		DetailsSchemaID:        "runecode.protocol.details.policy.evaluation.v0",
		Details:                map[string]any{"precedence": "test_seed"},
	}
	if err := service.RecordPolicyDecision("", "", decision); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
}

func writeApprovalFixtures(t *testing.T, approver string, digest string, repoPath string, commit string, extractorVersion string) (string, string, []trustpolicy.VerifierRecord) {
	t.Helper()
	requestEnvelope, decisionEnvelope, verifiers := signedApprovalArtifactsForCLITests(t, approver, digest, repoPath, commit, extractorVersion)
	approvalRequestPath := filepath.Join(t.TempDir(), "approval-request.json")
	approvalEnvelopePath := filepath.Join(t.TempDir(), "approval-envelope.json")
	approvalRequestJSON, err := json.Marshal(requestEnvelope)
	if err != nil {
		t.Fatalf("Marshal request envelope error: %v", err)
	}
	if err := os.WriteFile(approvalRequestPath, approvalRequestJSON, 0o600); err != nil {
		t.Fatalf("Write request envelope fixture error: %v", err)
	}
	approvalEnvelopeJSON, err := json.Marshal(decisionEnvelope)
	if err != nil {
		t.Fatalf("Marshal decision envelope error: %v", err)
	}
	if err := os.WriteFile(approvalEnvelopePath, approvalEnvelopeJSON, 0o600); err != nil {
		t.Fatalf("Write envelope fixture error: %v", err)
	}
	return approvalRequestPath, approvalEnvelopePath, verifiers
}

func signedApprovalArtifactsForCLITests(t *testing.T, approver string, digest string, repoPath string, commit string, extractorVersion string) (*trustpolicy.SignedObjectEnvelope, *trustpolicy.SignedObjectEnvelope, []trustpolicy.VerifierRecord) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey error: %v", err)
	}
	now := time.Now().UTC()
	request := approvalRequestFixture(approver, digest, repoPath, commit, extractorVersion, now)
	requestPayload, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal request error: %v", err)
	}
	canonicalRequest, err := jsoncanonicalizer.Transform(requestPayload)
	if err != nil {
		t.Fatalf("request canonicalization error: %v", err)
	}
	requestSig := ed25519.Sign(privateKey, canonicalRequest)
	keyIDValue := sha256Hex(publicKey)
	requestEnvelope := signedApprovalRequestEnvelopeFixture(requestPayload, keyIDValue, requestSig)
	requestHash, err := canonicalJSONDigest(requestPayload)
	if err != nil {
		t.Fatalf("request digest error: %v", err)
	}

	decision := approvalDecisionFixture(approver, requestHash, now)
	decisionPayload, err := json.Marshal(decision)
	if err != nil {
		t.Fatalf("Marshal decision error: %v", err)
	}
	canonicalDecision, err := jsoncanonicalizer.Transform(decisionPayload)
	if err != nil {
		t.Fatalf("decision canonicalization error: %v", err)
	}
	decisionSig := ed25519.Sign(privateKey, canonicalDecision)
	envelope := signedApprovalEnvelopeFixture(decisionPayload, keyIDValue, decisionSig)
	verifiers := []trustpolicy.VerifierRecord{approvalVerifierFixture(approver, keyIDValue, publicKey)}
	return requestEnvelope, envelope, verifiers
}

func approvalDecisionFixture(approver string, requestHash string, now time.Time) map[string]any {
	hashAlg, hash := splitDigestIdentity(requestHash)
	return map[string]any{"schema_id": trustpolicy.ApprovalDecisionSchemaID, "schema_version": trustpolicy.ApprovalDecisionSchemaVersion, "approval_request_hash": map[string]any{"hash_alg": hashAlg, "hash": hash}, "approver": map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "user", "principal_id": approver, "instance_id": "approval-session"}, "decision_outcome": "approve", "approval_assurance_level": "reauthenticated", "presence_mode": "hardware_touch", "key_protection_posture": "hardware_backed", "identity_binding_posture": "attested", "approval_assertion_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}, "decided_at": now.Format(time.RFC3339), "consumption_posture": "single_use", "signatures": []any{approvalDecisionSignaturePlaceholder()}}
}

func approvalRequestFixture(approver string, digest string, repoPath string, commit string, extractorVersion string, now time.Time) map[string]any {
	actionHash, err := artifacts.CanonicalPromotionActionRequestHash(artifacts.PromotionRequest{
		UnapprovedDigest:     digest,
		Approver:             approver,
		RepoPath:             repoPath,
		Commit:               commit,
		ExtractorToolVersion: extractorVersion,
	})
	if err != nil {
		panic(err)
	}
	actionHashAlg, actionHashValue := splitDigestIdentity(actionHash)
	digestHashAlg, digestHashValue := splitDigestIdentity(digest)
	return map[string]any{"schema_id": trustpolicy.ApprovalRequestSchemaID, "schema_version": trustpolicy.ApprovalRequestSchemaVersion, "approval_profile": "moderate", "requester": map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-artifact-store"}, "approval_trigger_code": "excerpt_promotion", "manifest_hash": map[string]any{"hash_alg": digestHashAlg, "hash": digestHashValue}, "action_request_hash": map[string]any{"hash_alg": actionHashAlg, "hash": actionHashValue}, "relevant_artifact_hashes": []any{map[string]any{"hash_alg": digestHashAlg, "hash": digestHashValue}}, "details_schema_id": "runecode.protocol.details.approval.excerpt-promotion.v0", "details": map[string]any{"repo_path": repoPath, "commit": commit}, "approval_assurance_level": "reauthenticated", "presence_mode": "hardware_touch", "requested_at": now.Add(-1 * time.Minute).Format(time.RFC3339), "expires_at": now.Add(30 * time.Minute).Format(time.RFC3339), "staleness_posture": "invalidate_on_bound_input_change", "changes_if_approved": "Promote reviewed file excerpts for downstream use.", "signatures": []any{approvalDecisionSignaturePlaceholder()}}
}

func approvalDecisionSignaturePlaceholder() map[string]any {
	return map[string]any{"alg": "ed25519", "key_id": trustpolicy.KeyIDProfile, "key_id_value": strings.Repeat("a", 64), "signature": "c2ln"}
}

func signedApprovalEnvelopeFixture(payload []byte, keyIDValue string, signature []byte) *trustpolicy.SignedObjectEnvelope {
	return &trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.ApprovalDecisionSchemaID, PayloadSchemaVersion: trustpolicy.ApprovalDecisionSchemaVersion, Payload: payload, SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Signature: base64.StdEncoding.EncodeToString(signature)}}
}

func signedApprovalRequestEnvelopeFixture(payload []byte, keyIDValue string, signature []byte) *trustpolicy.SignedObjectEnvelope {
	return &trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.ApprovalRequestSchemaID, PayloadSchemaVersion: trustpolicy.ApprovalRequestSchemaVersion, Payload: payload, SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Signature: base64.StdEncoding.EncodeToString(signature)}}
}

func approvalVerifierFixture(approver string, keyIDValue string, publicKey []byte) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{SchemaID: trustpolicy.VerifierSchemaID, SchemaVersion: trustpolicy.VerifierSchemaVersion, KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Alg: "ed25519", PublicKey: trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)}, LogicalPurpose: "approval_authority", LogicalScope: "user", OwnerPrincipal: trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "user", PrincipalID: approver, InstanceID: "approval-session"}, KeyProtectionPosture: "hardware_backed", IdentityBindingPosture: "attested", PresenceMode: "hardware_touch", CreatedAt: "2026-03-13T12:00:00Z", Status: "active"}
}

func canonicalJSONDigest(payload []byte) (string, error) {
	canonical, err := jsoncanonicalizer.Transform(payload)
	if err != nil {
		return "", err
	}
	return "sha256:" + sha256Hex(canonical), nil
}

func splitDigestIdentity(identity string) (string, string) {
	parts := strings.SplitN(identity, ":", 2)
	if len(parts) != 2 {
		return "sha256", identity
	}
	return parts[0], parts[1]
}

func seedTrustedVerifierForBrokerCLITest(t *testing.T, verifiers []trustpolicy.VerifierRecord) {
	t.Helper()
	for index := range verifiers {
		payload, err := json.Marshal(verifiers[index])
		if err != nil {
			t.Fatalf("Marshal verifier record error: %v", err)
		}
		verifierPath := filepath.Join(t.TempDir(), fmt.Sprintf("verifier-%d.json", index))
		if err := os.WriteFile(verifierPath, payload, 0o600); err != nil {
			t.Fatalf("WriteFile verifier record error: %v", err)
		}
		evidencePath := writeTrustedImportEvidenceFixture(t, "verifier-record")
		if err := run([]string{"import-trusted-contract", "--kind", "verifier-record", "--file", verifierPath, "--evidence", evidencePath}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
			t.Fatalf("import-trusted-contract returned error: %v", err)
		}
	}
}

func approvalRequestDigestForCLITests(envelope trustpolicy.SignedObjectEnvelope) (string, error) {
	canonical, err := jsoncanonicalizer.Transform(envelope.Payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func mustTimeForCLITests(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("time.Parse returned error for %q: %v", value, err)
	}
	return parsed
}

func stringField(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return value
}

func digestField(payload map[string]any, key string) string {
	obj, _ := payload[key].(map[string]any)
	hashAlg, _ := obj["hash_alg"].(string)
	hash, _ := obj["hash"].(string)
	if hashAlg == "" || hash == "" {
		return ""
	}
	return hashAlg + ":" + hash
}

func digestListField(payload map[string]any, key string) []string {
	items, _ := payload[key].([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		obj, _ := item.(map[string]any)
		hashAlg, _ := obj["hash_alg"].(string)
		hash, _ := obj["hash"].(string)
		if hashAlg != "" && hash != "" {
			out = append(out, hashAlg+":"+hash)
		}
	}
	return out
}

func writeTrustedImportEvidenceFixture(t *testing.T, kind string) string {
	t.Helper()
	evidence := map[string]any{
		"schema_id":      "runecode.protocol.v0.TrustedContractImportRequest",
		"schema_version": "0.1.0",
		"kind":           kind,
		"importer": map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "user",
			"principal_id":   "operator",
			"instance_id":    "cli-session",
		},
		"reason":      "manual import for verifier rotation",
		"imported_at": "2026-04-08T00:00:00Z",
		"source":      "local-trust-bundle",
	}
	b, err := json.Marshal(evidence)
	if err != nil {
		t.Fatalf("Marshal trusted import evidence error: %v", err)
	}
	path := filepath.Join(t.TempDir(), "import-evidence.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("Write trusted import evidence error: %v", err)
	}
	return path
}

func sha256Hex(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func setBrokerServiceForTest(t *testing.T) string {
	t.Helper()
	root := filepath.Join(canonicalTempDir(t), "store")
	secretsRoot := filepath.Join(root, "secrets-state")
	seedBrokerSecretsReadinessState(t, secretsRoot)
	t.Setenv("RUNE_SECRETS_STATE_ROOT", secretsRoot)
	originalResolver := localAPIClientModeResolver
	localAPIClientModeResolver = func() (brokerLocalAPIClientFactory, error) {
		return func(service *brokerapi.Service) brokerLocalAPI {
			if service == nil {
				resolvedService, err := brokerServiceFactory(defaultBrokerServiceRoots())
				if err != nil {
					return newUnavailableLocalAPIClient(err)
				}
				service = resolvedService
			}
			return newInProcessLocalAPIClient(service)
		}, nil
	}
	brokerServiceFactory = func(_ brokerServiceRoots) (*brokerapi.Service, error) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
		return brokerapi.NewServiceWithConfig(root, filepath.Join(root, "audit-ledger"), brokerapi.APIConfig{RepositoryRoot: repoRoot})
	}
	t.Cleanup(func() {
		brokerServiceFactory = newBrokerService
		localAPIClientModeResolver = originalResolver
	})
	return root
}

func seedBrokerSecretsReadinessState(t *testing.T, root string) {
	t.Helper()
	svc, err := secretsd.Open(root)
	if err != nil {
		t.Fatalf("secretsd.Open returned error: %v", err)
	}
	if _, err := svc.ImportSecret("secrets/prod/db", strings.NewReader("db-secret")); err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
	lease, err := svc.IssueLease(secretsd.IssueLeaseRequest{SecretRef: "secrets/prod/db", ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", TTLSeconds: 120})
	if err != nil {
		t.Fatalf("IssueLease returned error: %v", err)
	}
	if _, err := svc.RenewLease(secretsd.RenewLeaseRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", TTLSeconds: 120}); err != nil {
		t.Fatalf("RenewLease returned error: %v", err)
	}
	if _, err := svc.RevokeLease(secretsd.RevokeLeaseRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", Reason: "operator"}); err != nil {
		t.Fatalf("RevokeLease returned error: %v", err)
	}
}

func canonicalTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) returned error: %v", dir, err)
	}
	return resolved
}

func testDigest(seed string) string {
	base := strings.Repeat(seed, 64)
	if len(base) > 64 {
		base = base[:64]
	}
	for len(base) < 64 {
		base += "0"
	}
	return "sha256:" + base
}
