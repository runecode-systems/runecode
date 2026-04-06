package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func TestHelpAndUnknownCommand(t *testing.T) {
	setBrokerServiceForTest(t)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := run([]string{"--help"}, stdout, stderr); err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage: runecode-broker") {
		t.Fatalf("help output missing usage: %q", stdout.String())
	}
	err := run([]string{"not-a-command"}, stdout, stderr)
	if err == nil {
		t.Fatal("expected usage error for unknown command")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("unknown command error type = %T, want *usageError", err)
	}
}

func TestPutListHeadGetArtifactCLI(t *testing.T) {
	root := setBrokerServiceForTest(t)
	payloadPath := writeTempFile(t, "payload.txt", "hello artifact")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ref := putArtifactViaCLI(t, stdout, stderr, payloadPath, "spec_text", testDigest("1"))
	list := listArtifactsViaCLI(t, stdout, stderr)
	if len(list) != 1 {
		t.Fatalf("list-artifacts count = %d, want 1", len(list))
	}
	record := headArtifactViaCLI(t, stdout, stderr, ref.Digest)
	if record.Reference.Digest != ref.Digest {
		t.Fatalf("head digest = %q, want %q", record.Reference.Digest, ref.Digest)
	}
	outputPath := filepath.Join(t.TempDir(), "output.txt")
	getArtifactViaCLI(t, stdout, stderr, ref.Digest, outputPath)
	b, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("read get-artifact output error: %v", readErr)
	}
	if string(b) != "hello artifact" {
		t.Fatalf("get-artifact payload = %q, want hello artifact", string(b))
	}

	if _, err := os.Stat(filepath.Join(root, "state.json")); err != nil {
		t.Fatalf("expected broker state.json: %v", err)
	}
}

func TestPromotionFlowAndCheckFlowCLI(t *testing.T) {
	setBrokerServiceForTest(t)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	unapprovedPath := writeTempFile(t, "excerpt.txt", "private excerpt")
	unapproved := putArtifactViaCLI(t, stdout, stderr, unapprovedPath, "unapproved_file_excerpts", testDigest("2"))
	approvalRequestPath, approvalEnvelopePath, verifierRecords := writeApprovalFixtures(t, "human", unapproved.Digest, "repo/file.txt", "abc123", "tool-v1")
	seedTrustedVerifierForBrokerCLITest(t, verifierRecords)
	err := run([]string{"check-flow", "--producer", "workspace", "--consumer", "model_gateway", "--data-class", "unapproved_file_excerpts", "--digest", unapproved.Digest, "--egress"}, stdout, stderr)
	if err != artifacts.ErrUnapprovedEgressDenied {
		t.Fatalf("check-flow unapproved egress error = %v, want %v", err, artifacts.ErrUnapprovedEgressDenied)
	}
	approved := promoteViaCLI(t, stdout, stderr, unapproved.Digest, approvalRequestPath, approvalEnvelopePath)
	err = run([]string{"check-flow", "--producer", "workspace", "--consumer", "model_gateway", "--data-class", "approved_file_excerpts", "--digest", approved.Digest, "--egress"}, stdout, stderr)
	if err != artifacts.ErrApprovedEgressRequiresManifest {
		t.Fatalf("check-flow approved no opt-in error = %v, want %v", err, artifacts.ErrApprovedEgressRequiresManifest)
	}
	err = run([]string{"check-flow", "--producer", "workspace", "--consumer", "model_gateway", "--data-class", "approved_file_excerpts", "--digest", approved.Digest, "--egress", "--manifest-opt-in"}, stdout, stderr)
	if err != nil {
		t.Fatalf("check-flow approved with opt-in error: %v", err)
	}
}

func TestGCAndBackupCommands(t *testing.T) {
	setBrokerServiceForTest(t)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	payloadPath := writeTempFile(t, "tmp.txt", "tmp payload")
	err := run([]string{"put-artifact", "--file", payloadPath, "--content-type", "text/plain", "--data-class", "spec_text", "--provenance-hash", testDigest("3"), "--run-id", "run-1"}, stdout, stderr)
	if err != nil {
		t.Fatalf("put-artifact returned error: %v", err)
	}
	err = run([]string{"set-run-status", "--run-id", "run-1", "--status", "closed"}, stdout, stderr)
	if err != nil {
		t.Fatalf("set-run-status returned error: %v", err)
	}
	stdout.Reset()
	err = run([]string{"gc"}, stdout, stderr)
	if err != nil {
		t.Fatalf("gc returned error: %v", err)
	}
	result := artifacts.GCResult{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &result); unmarshalErr != nil {
		t.Fatalf("gc output parse error: %v", unmarshalErr)
	}

	backupPath := filepath.Join(t.TempDir(), "artifact-backup.json")
	err = run([]string{"export-backup", "--path", backupPath}, stdout, stderr)
	if err != nil {
		t.Fatalf("export-backup returned error: %v", err)
	}
	err = run([]string{"restore-backup", "--path", backupPath}, stdout, stderr)
	if err != nil {
		t.Fatalf("restore-backup returned error: %v", err)
	}
}

func TestAuditReadinessAndVerificationCommands(t *testing.T) {
	root := setBrokerServiceForTest(t)
	if err := seedLedgerForBrokerCommandTest(filepath.Join(root, "audit-ledger")); err != nil {
		t.Fatalf("seedLedgerForBrokerCommandTest returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	if err := run([]string{"audit-readiness"}, stdout, stderr); err != nil {
		t.Fatalf("audit-readiness returned error: %v", err)
	}
	readiness := trustpolicy.AuditdReadiness{}
	if err := json.Unmarshal(stdout.Bytes(), &readiness); err != nil {
		t.Fatalf("audit-readiness output parse error: %v", err)
	}
	if !readiness.Ready {
		t.Fatal("readiness.ready = false, want true")
	}

	stdout.Reset()
	if err := run([]string{"audit-verification", "--limit", "5"}, stdout, stderr); err != nil {
		t.Fatalf("audit-verification returned error: %v", err)
	}
	surface := brokerapi.AuditVerificationSurface{}
	if err := json.Unmarshal(stdout.Bytes(), &surface); err != nil {
		t.Fatalf("audit-verification output parse error: %v", err)
	}
	if len(surface.Views) == 0 {
		t.Fatal("audit-verification views empty, want default operational view entries")
	}
}

func writeTempFile(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write temp file error: %v", err)
	}
	return path
}

func putArtifactViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, path, dataClass, provenance string) artifacts.ArtifactReference {
	t.Helper()
	stdout.Reset()
	err := run([]string{"put-artifact", "--file", path, "--content-type", "text/plain", "--data-class", dataClass, "--provenance-hash", provenance}, stdout, stderr)
	if err != nil {
		t.Fatalf("put-artifact returned error: %v", err)
	}
	ref := artifacts.ArtifactReference{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &ref); unmarshalErr != nil {
		t.Fatalf("put-artifact output parse error: %v", unmarshalErr)
	}
	return ref
}

func listArtifactsViaCLI(t *testing.T, stdout, stderr *bytes.Buffer) []artifacts.ArtifactRecord {
	t.Helper()
	stdout.Reset()
	err := run([]string{"list-artifacts"}, stdout, stderr)
	if err != nil {
		t.Fatalf("list-artifacts returned error: %v", err)
	}
	list := []artifacts.ArtifactRecord{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &list); unmarshalErr != nil {
		t.Fatalf("list-artifacts output parse error: %v", unmarshalErr)
	}
	return list
}

func headArtifactViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, digest string) artifacts.ArtifactRecord {
	t.Helper()
	stdout.Reset()
	err := run([]string{"head-artifact", "--digest", digest}, stdout, stderr)
	if err != nil {
		t.Fatalf("head-artifact returned error: %v", err)
	}
	record := artifacts.ArtifactRecord{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &record); unmarshalErr != nil {
		t.Fatalf("head-artifact output parse error: %v", unmarshalErr)
	}
	return record
}

func getArtifactViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, digest, out string) {
	t.Helper()
	stdout.Reset()
	err := run([]string{"get-artifact", "--digest", digest, "--out", out}, stdout, stderr)
	if err != nil {
		t.Fatalf("get-artifact returned error: %v", err)
	}
}

func promoteViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, digest string, approvalRequestPath string, approvalEnvelopePath string) artifacts.ArtifactReference {
	t.Helper()
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
	request := approvalRequestFixture(approver, digest, repoPath, commit, extractorVersion)
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

	decision := approvalDecisionFixture(approver, requestHash)
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

func approvalDecisionFixture(approver string, requestHash string) map[string]any {
	hashAlg, hash := splitDigestIdentity(requestHash)
	return map[string]any{
		"schema_id":                trustpolicy.ApprovalDecisionSchemaID,
		"schema_version":           trustpolicy.ApprovalDecisionSchemaVersion,
		"approval_request_hash":    map[string]any{"hash_alg": hashAlg, "hash": hash},
		"approver":                 map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "user", "principal_id": approver, "instance_id": "approval-session"},
		"decision_outcome":         "approve",
		"approval_assurance_level": "reauthenticated",
		"presence_mode":            "hardware_touch",
		"key_protection_posture":   "hardware_backed",
		"identity_binding_posture": "attested",
		"approval_assertion_hash":  map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)},
		"decided_at":               "2026-03-13T12:05:00Z",
		"consumption_posture":      "single_use",
		"signatures":               []any{approvalDecisionSignaturePlaceholder()},
	}
}

func approvalRequestFixture(approver string, digest string, repoPath string, commit string, extractorVersion string) map[string]any {
	actionHash := promotionActionHashForCLITests(digest, repoPath, commit, extractorVersion, approver)
	actionHashAlg, actionHashValue := splitDigestIdentity(actionHash)
	digestHashAlg, digestHashValue := splitDigestIdentity(digest)
	return map[string]any{
		"schema_id":                trustpolicy.ApprovalRequestSchemaID,
		"schema_version":           trustpolicy.ApprovalRequestSchemaVersion,
		"approval_profile":         "moderate",
		"requester":                map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-artifact-store"},
		"approval_trigger_code":    "artifact_promotion",
		"manifest_hash":            map[string]any{"hash_alg": digestHashAlg, "hash": digestHashValue},
		"action_request_hash":      map[string]any{"hash_alg": actionHashAlg, "hash": actionHashValue},
		"relevant_artifact_hashes": []any{map[string]any{"hash_alg": digestHashAlg, "hash": digestHashValue}},
		"details_schema_id":        "runecode.protocol.details.approval.excerpt-promotion.v0",
		"details":                  map[string]any{"repo_path": repoPath, "commit": commit},
		"approval_assurance_level": "reauthenticated",
		"presence_mode":            "hardware_touch",
		"requested_at":             "2026-03-13T12:00:00Z",
		"expires_at":               "2026-03-13T12:30:00Z",
		"staleness_posture":        "invalidate_on_bound_input_change",
		"changes_if_approved":      "Promote reviewed file excerpts for downstream use.",
		"signatures":               []any{approvalDecisionSignaturePlaceholder()},
	}
}

func approvalDecisionSignaturePlaceholder() map[string]any {
	return map[string]any{
		"alg":          "ed25519",
		"key_id":       trustpolicy.KeyIDProfile,
		"key_id_value": strings.Repeat("a", 64),
		"signature":    "c2ln",
	}
}

func signedApprovalEnvelopeFixture(payload []byte, keyIDValue string, signature []byte) *trustpolicy.SignedObjectEnvelope {
	return &trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.ApprovalDecisionSchemaID,
		PayloadSchemaVersion: trustpolicy.ApprovalDecisionSchemaVersion,
		Payload:              payload,
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature: trustpolicy.SignatureBlock{
			Alg:        "ed25519",
			KeyID:      trustpolicy.KeyIDProfile,
			KeyIDValue: keyIDValue,
			Signature:  base64.StdEncoding.EncodeToString(signature),
		},
	}
}

func signedApprovalRequestEnvelopeFixture(payload []byte, keyIDValue string, signature []byte) *trustpolicy.SignedObjectEnvelope {
	return &trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.ApprovalRequestSchemaID,
		PayloadSchemaVersion: trustpolicy.ApprovalRequestSchemaVersion,
		Payload:              payload,
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature: trustpolicy.SignatureBlock{
			Alg:        "ed25519",
			KeyID:      trustpolicy.KeyIDProfile,
			KeyIDValue: keyIDValue,
			Signature:  base64.StdEncoding.EncodeToString(signature),
		},
	}
}

func approvalVerifierFixture(approver string, keyIDValue string, publicKey []byte) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             keyIDValue,
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)},
		LogicalPurpose:         "approval_authority",
		LogicalScope:           "user",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "user", PrincipalID: approver, InstanceID: "approval-session"},
		KeyProtectionPosture:   "hardware_backed",
		IdentityBindingPosture: "attested",
		PresenceMode:           "hardware_touch",
		CreatedAt:              "2026-03-13T12:00:00Z",
		Status:                 "active",
	}
}

func TestPromoteExcerptRequiresSignedApprovalInputs(t *testing.T) {
	setBrokerServiceForTest(t)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	unapprovedPath := writeTempFile(t, "excerpt.txt", "private excerpt")
	unapproved := putArtifactViaCLI(t, stdout, stderr, unapprovedPath, "unapproved_file_excerpts", testDigest("2"))
	err := run([]string{"promote-excerpt", "--unapproved-digest", unapproved.Digest, "--approver", "human", "--repo-path", "repo/file.txt", "--commit", "abc123", "--extractor-version", "tool-v1", "--full-content-visible"}, stdout, stderr)
	if err == nil {
		t.Fatal("promote-excerpt expected usage error when signed approval inputs are missing")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
}

func TestPromoteExcerptRejectsSelfProvidedVerifierRecord(t *testing.T) {
	setBrokerServiceForTest(t)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	unapprovedPath := writeTempFile(t, "excerpt.txt", "private excerpt")
	unapproved := putArtifactViaCLI(t, stdout, stderr, unapprovedPath, "unapproved_file_excerpts", testDigest("2"))
	approvalRequestPath, approvalEnvelopePath, _ := writeApprovalFixtures(t, "human", unapproved.Digest, "repo/file.txt", "abc123", "tool-v1")
	_, _, verifierRecords := signedApprovalArtifactsForCLITests(t, "human", unapproved.Digest, "repo/file.txt", "abc123", "tool-v1")
	for index := range verifierRecords {
		payload, err := json.Marshal(verifierRecords[index])
		if err != nil {
			t.Fatalf("Marshal verifier error: %v", err)
		}
		payloadPath := writeTempFile(t, "verifier-non-auditd.json", string(payload))
		nibble := string('a' + rune(index%6))
		err = run([]string{"put-artifact", "--file", payloadPath, "--content-type", "application/json", "--data-class", "audit_verification_report", "--provenance-hash", testDigest(nibble), "--role", "workspace"}, stdout, stderr)
		if err != nil {
			t.Fatalf("put-artifact verifier record returned error: %v", err)
		}
	}
	err := run([]string{"promote-excerpt", "--unapproved-digest", unapproved.Digest, "--approver", "human", "--approval-request", approvalRequestPath, "--approval-envelope", approvalEnvelopePath, "--repo-path", "repo/file.txt", "--commit", "abc123", "--extractor-version", "tool-v1", "--full-content-visible"}, stdout, stderr)
	if err == nil {
		t.Fatal("promote-excerpt expected error when verifier records are not auditd-owned")
	}
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

func promotionActionHashForCLITests(digest string, repoPath string, commit string, extractorVersion string, approver string) string {
	payload, err := json.Marshal(struct {
		Approver             string `json:"approver"`
		Commit               string `json:"commit"`
		ExtractorToolVersion string `json:"extractor_tool_version"`
		RepoPath             string `json:"repo_path"`
		UnapprovedDigest     string `json:"unapproved_digest"`
	}{
		Approver:             approver,
		Commit:               commit,
		ExtractorToolVersion: extractorVersion,
		RepoPath:             repoPath,
		UnapprovedDigest:     digest,
	})
	if err != nil {
		panic(err)
	}
	canonical, err := jsoncanonicalizer.Transform(payload)
	if err != nil {
		panic(err)
	}
	return "sha256:" + sha256Hex(canonical)
}

func seedTrustedVerifierForBrokerCLITest(t *testing.T, verifiers []trustpolicy.VerifierRecord) {
	t.Helper()
	for index := range verifiers {
		if err := putTrustedVerifierRecordForTest(verifiers[index]); err != nil {
			t.Fatalf("put trusted verifier record returned error: %v", err)
		}
	}
}

func putTrustedVerifierRecordForTest(record trustpolicy.VerifierRecord) error {
	service, err := brokerServiceFactory()
	if err != nil {
		return err
	}
	return putTrustedVerifierRecord(service, record)
}

func sha256Hex(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func setBrokerServiceForTest(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "store")
	brokerServiceFactory = func() (*brokerapi.Service, error) {
		return brokerapi.NewService(root, filepath.Join(root, "audit-ledger"))
	}
	t.Cleanup(func() {
		brokerServiceFactory = brokerService
	})
	return root
}

func TestBrokerServiceUsesTempFallbackWhenUserDirsUnavailable(t *testing.T) {
	originalFactory := brokerServiceFactory
	defer func() { brokerServiceFactory = originalFactory }()

	t.Setenv("HOME", "")
	if err := os.Unsetenv("XDG_CACHE_HOME"); err != nil {
		t.Fatalf("Unsetenv(XDG_CACHE_HOME) error: %v", err)
	}
	if err := os.Unsetenv("XDG_CONFIG_HOME"); err != nil {
		t.Fatalf("Unsetenv(XDG_CONFIG_HOME) error: %v", err)
	}

	root := defaultBrokerStoreRoot()
	if root == "" {
		t.Fatal("defaultBrokerStoreRoot returned empty path")
	}
	if !filepath.IsAbs(root) {
		t.Fatalf("defaultBrokerStoreRoot = %q, want absolute path", root)
	}
	if !strings.Contains(filepath.ToSlash(root), "/runecode/artifact-store") {
		t.Fatalf("defaultBrokerStoreRoot = %q, want path containing runecode/artifact-store", root)
	}
	if _, err := brokerapi.NewService(root, filepath.Join(root, "audit-ledger")); err != nil {
		t.Fatalf("NewService(%q) error: %v", root, err)
	}
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

func seedLedgerForBrokerCommandTest(root string) error {
	if err := seedLedgerDirectoriesForBrokerTest(root); err != nil {
		return err
	}
	recordDigest, canonicalEnvelope, err := seedEventRecordForBrokerTest()
	if err != nil {
		return err
	}
	if err := seedSegmentsForBrokerTest(root, recordDigest, canonicalEnvelope); err != nil {
		return err
	}
	sealID, err := seedSealForBrokerTest(root, recordDigest)
	if err != nil {
		return err
	}
	reportID, err := seedVerificationReportForBrokerTest(root)
	if err != nil {
		return err
	}
	if err := seedStateForBrokerTest(root, sealID, reportID); err != nil {
		return err
	}
	return seedContractsForBrokerTest(root)
}

func seedLedgerDirectoriesForBrokerTest(root string) error {
	paths := []string{
		filepath.Join(root, "segments"),
		filepath.Join(root, "sidecar", "segment-seals"),
		filepath.Join(root, "sidecar", "verification-reports"),
		filepath.Join(root, "contracts"),
	}
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func seedEventRecordForBrokerTest() (trustpolicy.Digest, []byte, error) {
	eventPayload := map[string]any{"session_id": "session-1"}
	eventPayloadBytes, _ := json.Marshal(eventPayload)
	canonicalEventPayload, _ := jsoncanonicalizer.Transform(eventPayloadBytes)
	eventPayloadHash := sha256.Sum256(canonicalEventPayload)
	event := map[string]any{
		"schema_id":                     trustpolicy.AuditEventSchemaID,
		"schema_version":                trustpolicy.AuditEventSchemaVersion,
		"audit_event_type":              "isolate_session_bound",
		"emitter_stream_id":             "auditd-stream-1",
		"seq":                           1,
		"occurred_at":                   "2026-03-13T12:15:00Z",
		"principal":                     map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "auditd", "instance_id": "auditd-1"},
		"event_payload_schema_id":       "runecode.protocol.audit.payload.isolate-session-bound.v0",
		"event_payload":                 eventPayload,
		"event_payload_hash":            map[string]any{"hash_alg": "sha256", "hash": hex.EncodeToString(eventPayloadHash[:])},
		"protocol_bundle_manifest_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("b", 64)},
		"scope":                         map[string]any{"workspace_id": "workspace-1", "run_id": "run-1", "stage_id": "stage-1"},
		"correlation":                   map[string]any{"session_id": "session-1", "operation_id": "op-1"},
		"subject_ref":                   map[string]any{"object_family": "isolate_binding", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("c", 64)}, "ref_role": "binding_target"},
		"cause_refs":                    []any{map[string]any{"object_family": "audit_event", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("d", 64)}, "ref_role": "session_cause"}},
		"related_refs":                  []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("e", 64)}, "ref_role": "binding"}},
		"signer_evidence_refs":          []any{map[string]any{"object_family": "verifier_record", "digest": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}, "ref_role": "admissibility"}},
	}
	envelope := trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.AuditEventSchemaID, PayloadSchemaVersion: trustpolicy.AuditEventSchemaVersion, Payload: mustJSONMarshalBrokerTest(event), SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: strings.Repeat("a", 64), Signature: base64.StdEncoding.EncodeToString(make([]byte, 64))}}
	envelopeBytes, _ := json.Marshal(envelope)
	canonicalEnvelope, _ := jsoncanonicalizer.Transform(envelopeBytes)
	recordSum := sha256.Sum256(canonicalEnvelope)
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(recordSum[:])}, canonicalEnvelope, nil
}

func seedSegmentsForBrokerTest(root string, recordDigest trustpolicy.Digest, canonicalEnvelope []byte) error {
	sealed := trustpolicy.AuditSegmentFilePayload{SchemaID: "runecode.protocol.v0.AuditSegmentFile", SchemaVersion: "0.1.0", Header: trustpolicy.AuditSegmentHeader{Format: "audit_segment_framed_v1", SegmentID: "segment-000001", SegmentState: trustpolicy.AuditSegmentStateSealed, CreatedAt: "2026-03-13T12:00:00Z", Writer: "auditd"}, Frames: []trustpolicy.AuditSegmentRecordFrame{{RecordDigest: recordDigest, ByteLength: int64(len(canonicalEnvelope)), CanonicalSignedEnvelopeBytes: base64.StdEncoding.EncodeToString(canonicalEnvelope)}}, LifecycleMarker: trustpolicy.AuditSegmentLifecycleMarker{State: trustpolicy.AuditSegmentStateSealed, MarkedAt: "2026-03-13T12:20:00Z"}}
	open := trustpolicy.AuditSegmentFilePayload{SchemaID: "runecode.protocol.v0.AuditSegmentFile", SchemaVersion: "0.1.0", Header: trustpolicy.AuditSegmentHeader{Format: "audit_segment_framed_v1", SegmentID: "segment-000002", SegmentState: trustpolicy.AuditSegmentStateOpen, CreatedAt: "2026-03-13T12:21:00Z", Writer: "auditd"}, Frames: []trustpolicy.AuditSegmentRecordFrame{}, LifecycleMarker: trustpolicy.AuditSegmentLifecycleMarker{State: trustpolicy.AuditSegmentStateOpen, MarkedAt: "2026-03-13T12:21:00Z"}}
	if err := writeCanonicalJSONForBrokerTest(filepath.Join(root, "segments", "segment-000001.json"), sealed); err != nil {
		return err
	}
	return writeCanonicalJSONForBrokerTest(filepath.Join(root, "segments", "segment-000002.json"), open)
}

func seedSealForBrokerTest(root string, recordDigest trustpolicy.Digest) (string, error) {
	seal := trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.AuditSegmentSealSchemaID, PayloadSchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion, Payload: mustJSONMarshalBrokerTest(trustpolicy.AuditSegmentSealPayload{SchemaID: trustpolicy.AuditSegmentSealSchemaID, SchemaVersion: trustpolicy.AuditSegmentSealSchemaVersion, SegmentID: "segment-000001", SealedAfterState: trustpolicy.AuditSegmentStateOpen, SegmentState: trustpolicy.AuditSegmentStateSealed, SegmentCut: trustpolicy.AuditSegmentCutWindowPolicy{OwnershipScope: trustpolicy.AuditSegmentOwnershipScopeInstanceGlobal, MaxSegmentBytes: 2048, CutTrigger: trustpolicy.AuditSegmentCutTriggerSizeWindow}, EventCount: 1, FirstRecordDigest: recordDigest, LastRecordDigest: recordDigest, MerkleProfile: trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1, MerkleRoot: recordDigest, SegmentFileHashScope: trustpolicy.AuditSegmentFileHashScopeRawFramedV1, SegmentFileHash: recordDigest, SealChainIndex: 0, AnchoringSubject: trustpolicy.AuditSegmentAnchoringSubjectSeal, SealedAt: "2026-03-13T12:20:00Z", ProtocolBundleManifestHash: trustpolicy.Digest{HashAlg: "sha256", Hash: strings.Repeat("b", 64)}, SealReason: "size_threshold"}), SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: strings.Repeat("a", 64), Signature: base64.StdEncoding.EncodeToString(make([]byte, 64))}}
	sealDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(seal)
	if err != nil {
		return "", err
	}
	sealID, _ := sealDigest.Identity()
	if err := writeCanonicalJSONForBrokerTest(filepath.Join(root, "sidecar", "segment-seals", strings.TrimPrefix(sealID, "sha256:")+".json"), seal); err != nil {
		return "", err
	}
	return sealID, nil
}

func seedVerificationReportForBrokerTest(root string) (string, error) {
	report := trustpolicy.AuditVerificationReportPayload{SchemaID: trustpolicy.AuditVerificationReportSchemaID, SchemaVersion: trustpolicy.AuditVerificationReportSchemaVersion, VerifiedAt: time.Now().UTC().Format(time.RFC3339), VerificationScope: trustpolicy.AuditVerificationScope{ScopeKind: trustpolicy.AuditVerificationScopeSegment, LastSegmentID: "segment-000001"}, CryptographicallyValid: true, HistoricallyAdmissible: true, CurrentlyDegraded: false, IntegrityStatus: trustpolicy.AuditVerificationStatusOK, AnchoringStatus: trustpolicy.AuditVerificationStatusOK, StoragePostureStatus: trustpolicy.AuditVerificationStatusOK, SegmentLifecycleStatus: trustpolicy.AuditVerificationStatusOK, DegradedReasons: []string{}, HardFailures: []string{}, Findings: []trustpolicy.AuditVerificationFinding{}, Summary: "ok"}
	reportCanonical, _ := jsoncanonicalizer.Transform(mustJSONMarshalBrokerTest(report))
	reportSum := sha256.Sum256(reportCanonical)
	reportDigest := trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(reportSum[:])}
	reportID, _ := reportDigest.Identity()
	if err := writeCanonicalJSONForBrokerTest(filepath.Join(root, "sidecar", "verification-reports", strings.TrimPrefix(reportID, "sha256:")+".json"), report); err != nil {
		return "", err
	}
	return reportID, nil
}

func seedStateForBrokerTest(root, sealID, reportID string) error {
	state := map[string]any{"schema_version": 1, "current_open_segment_id": "segment-000002", "next_segment_number": 3, "open_frame_count": 0, "last_seal_envelope_digest": sealID, "last_sealed_segment_id": "segment-000001", "last_verification_report_digest": reportID, "recovery_complete": true, "last_indexed_record_count": 1}
	return writeCanonicalJSONForBrokerTest(filepath.Join(root, "state.json"), state)
}

func seedContractsForBrokerTest(root string) error {
	publicKey := make([]byte, 32)
	keyID := sha256.Sum256(publicKey)
	verifier := trustpolicy.VerifierRecord{SchemaID: trustpolicy.VerifierSchemaID, SchemaVersion: trustpolicy.VerifierSchemaVersion, KeyID: trustpolicy.KeyIDProfile, KeyIDValue: hex.EncodeToString(keyID[:]), Alg: "ed25519", PublicKey: trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)}, LogicalPurpose: "isolate_session_identity", LogicalScope: "session", OwnerPrincipal: trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "auditd", InstanceID: "auditd-1"}, KeyProtectionPosture: "os_keystore", IdentityBindingPosture: "attested", PresenceMode: "os_confirmation", CreatedAt: "2026-03-13T12:00:00Z", Status: "active"}
	if err := writeCanonicalJSONForBrokerTest(filepath.Join(root, "contracts", "verifier-records.json"), []trustpolicy.VerifierRecord{verifier}); err != nil {
		return err
	}
	catalog := trustpolicy.AuditEventContractCatalog{SchemaID: trustpolicy.AuditEventContractCatalogSchemaID, SchemaVersion: trustpolicy.AuditEventContractCatalogSchemaVersion, CatalogID: "audit_event_contract_v0", Entries: []trustpolicy.AuditEventContractCatalogEntry{{AuditEventType: "isolate_session_bound", AllowedPayloadSchemaIDs: []string{"runecode.protocol.audit.payload.isolate-session-bound.v0"}, AllowedSignerPurposes: []string{"isolate_session_identity"}, AllowedSignerScopes: []string{"session"}, RequiredScopeFields: []string{"workspace_id", "run_id", "stage_id"}, RequiredCorrelationFields: []string{"session_id", "operation_id"}, RequireSubjectRef: true, AllowedSubjectRefRoles: []string{"binding_target"}, AllowedCauseRefRoles: []string{"session_cause"}, AllowedRelatedRefRoles: []string{"binding", "evidence", "receipt"}, RequireSignerEvidenceRefs: true, AllowedSignerEvidenceRefRoles: []string{"admissibility", "binding"}}}}
	return writeCanonicalJSONForBrokerTest(filepath.Join(root, "contracts", "event-contract-catalog.json"), catalog)
}

func writeCanonicalJSONForBrokerTest(path string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return err
	}
	return os.WriteFile(path, canonical, 0o600)
}

func mustJSONMarshalBrokerTest(value any) []byte {
	b, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return b
}
