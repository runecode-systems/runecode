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

func listArtifactsViaCLI(t *testing.T, stdout, stderr *bytes.Buffer) []artifacts.ArtifactReference {
	t.Helper()
	stdout.Reset()
	err := run([]string{"list-artifacts"}, stdout, stderr)
	if err != nil {
		t.Fatalf("list-artifacts returned error: %v", err)
	}
	list := []artifacts.ArtifactReference{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &list); unmarshalErr != nil {
		t.Fatalf("list-artifacts output parse error: %v", unmarshalErr)
	}
	return list
}

func headArtifactViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, digest string) artifacts.ArtifactReference {
	t.Helper()
	stdout.Reset()
	err := run([]string{"head-artifact", "--digest", digest}, stdout, stderr)
	if err != nil {
		t.Fatalf("head-artifact returned error: %v", err)
	}
	ref := artifacts.ArtifactReference{}
	if unmarshalErr := json.Unmarshal(stdout.Bytes(), &ref); unmarshalErr != nil {
		t.Fatalf("head-artifact output parse error: %v", unmarshalErr)
	}
	return ref
}

func getArtifactViaCLI(t *testing.T, stdout, stderr *bytes.Buffer, digest, producer, consumer, dataClass string, manifestOptIn bool, out string) {
	t.Helper()
	stdout.Reset()
	args := []string{"get-artifact", "--digest", digest, "--producer", producer, "--consumer", consumer, "--out", out}
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
	return map[string]any{"schema_id": trustpolicy.ApprovalDecisionSchemaID, "schema_version": trustpolicy.ApprovalDecisionSchemaVersion, "approval_request_hash": map[string]any{"hash_alg": hashAlg, "hash": hash}, "approver": map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "user", "principal_id": approver, "instance_id": "approval-session"}, "decision_outcome": "approve", "approval_assurance_level": "reauthenticated", "presence_mode": "hardware_touch", "key_protection_posture": "hardware_backed", "identity_binding_posture": "attested", "approval_assertion_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}, "decided_at": "2026-03-13T12:05:00Z", "consumption_posture": "single_use", "signatures": []any{approvalDecisionSignaturePlaceholder()}}
}

func approvalRequestFixture(approver string, digest string, repoPath string, commit string, extractorVersion string) map[string]any {
	actionHash := promotionActionHashForCLITests(digest, repoPath, commit, extractorVersion, approver)
	actionHashAlg, actionHashValue := splitDigestIdentity(actionHash)
	digestHashAlg, digestHashValue := splitDigestIdentity(digest)
	return map[string]any{"schema_id": trustpolicy.ApprovalRequestSchemaID, "schema_version": trustpolicy.ApprovalRequestSchemaVersion, "approval_profile": "moderate", "requester": map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-artifact-store"}, "approval_trigger_code": "excerpt_promotion", "manifest_hash": map[string]any{"hash_alg": digestHashAlg, "hash": digestHashValue}, "action_request_hash": map[string]any{"hash_alg": actionHashAlg, "hash": actionHashValue}, "relevant_artifact_hashes": []any{map[string]any{"hash_alg": digestHashAlg, "hash": digestHashValue}}, "details_schema_id": "runecode.protocol.details.approval.excerpt-promotion.v0", "details": map[string]any{"repo_path": repoPath, "commit": commit}, "approval_assurance_level": "reauthenticated", "presence_mode": "hardware_touch", "requested_at": "2026-03-13T12:00:00Z", "expires_at": "2026-03-13T12:30:00Z", "staleness_posture": "invalidate_on_bound_input_change", "changes_if_approved": "Promote reviewed file excerpts for downstream use.", "signatures": []any{approvalDecisionSignaturePlaceholder()}}
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

func promotionActionHashForCLITests(digest string, repoPath string, commit string, extractorVersion string, approver string) string {
	payload, err := json.Marshal(struct {
		Approver             string `json:"approver"`
		Commit               string `json:"commit"`
		ExtractorToolVersion string `json:"extractor_tool_version"`
		RepoPath             string `json:"repo_path"`
		UnapprovedDigest     string `json:"unapproved_digest"`
	}{Approver: approver, Commit: commit, ExtractorToolVersion: extractorVersion, RepoPath: repoPath, UnapprovedDigest: digest})
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
