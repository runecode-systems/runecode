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
		return brokerapi.NewService(root)
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
	if _, err := brokerapi.NewService(root); err != nil {
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
