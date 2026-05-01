package protocolschema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type fixtureManifestFile struct {
	SchemaFixtures         []schemaFixtureEntry         `json:"schema_fixtures"`
	StreamSequenceFixtures []streamSequenceFixtureEntry `json:"stream_sequence_fixtures"`
	RuntimeFixtures        []runtimeFixtureEntry        `json:"runtime_invariant_fixtures"`
	CanonicalFixtures      []canonicalFixtureEntry      `json:"canonicalization_fixtures"`
}

type schemaFixtureEntry struct {
	ID          string `json:"id"`
	SchemaPath  string `json:"schema_path"`
	FixturePath string `json:"fixture_path"`
	ExpectValid bool   `json:"expect_valid"`
}

type streamSequenceFixtureEntry struct {
	ID              string `json:"id"`
	EventSchemaPath string `json:"event_schema_path"`
	FixturePath     string `json:"fixture_path"`
	ExpectValid     bool   `json:"expect_valid"`
}

type runtimeFixtureEntry struct {
	ID          string `json:"id"`
	SchemaPath  string `json:"schema_path"`
	FixturePath string `json:"fixture_path"`
	Rule        string `json:"rule"`
	ExpectValid bool   `json:"expect_valid"`
}

type canonicalFixtureEntry struct {
	ID                string `json:"id"`
	PayloadPath       string `json:"payload_path"`
	CanonicalJSONPath string `json:"canonical_json_path"`
	SHA256            string `json:"sha256"`
	ExpectValid       bool   `json:"expect_valid"`
}

func fixtureRoot() string {
	return "../../protocol/fixtures"
}

func fixturePath(t *testing.T, rel string) string {
	t.Helper()
	return rootedSchemaPath(t, fixtureRoot(), rel, "protocol/fixtures")
}

func loadFixtureManifest(t *testing.T) fixtureManifestFile {
	t.Helper()

	var manifest fixtureManifestFile
	loadJSON(t, fixturePath(t, "manifest.json"), &manifest)
	return manifest
}

func loadJSONArray(t *testing.T, filePath string) []any {
	t.Helper()

	value := loadJSONValue(t, filePath)
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("Decode(%q) produced %T, want []any", filePath, value)
	}
	return items
}

func loadJSONValue(t *testing.T, filePath string) any {
	t.Helper()

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", filePath, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("Decode(%q) returned error: %v", filePath, err)
	}
	return value
}

func canonicalSHA256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func validateRuntimeInvariant(rule string, value map[string]any, manifest manifestFile, bundle compiledBundle) error {
	switch rule {
	case "llm_request_unique_artifact_digests":
		return requireUniqueArtifactDigests(value, "input_artifacts")
	case "llm_request_unique_tool_identities":
		return requireUniqueToolIdentities(value, "tool_allowlist")
	case "llm_response_unique_output_artifact_digests":
		return requireUniqueArtifactDigests(value, "output_artifacts")
	case "llm_response_unique_tool_call_ids":
		return requireUniqueToolCallIDs(value, "proposed_tool_calls")
	case "signed_envelope_payload_schema_match":
		return requireSignedEnvelopePayloadSchemaMatch(value, manifest, bundle)
	case "audit_receipt_import_restore_byte_identity":
		return requireImportRestoreReceiptByteIdentity(value)
	case "session_send_message_ack_alignment":
		return requireSessionSendMessageAckAlignment(value)
	case "external_anchor_evidence_conformance":
		return requireExternalAnchorEvidenceConformance(value)
	default:
		return fmt.Errorf("unknown runtime invariant rule %q", rule)
	}
}

func requireExternalAnchorEvidenceConformance(envelope map[string]any) error {
	const externalAnchorEvidenceSchemaID = "runecode.protocol.v0.ExternalAnchorEvidence"
	const externalAnchorEvidenceSchemaVersion = "0.1.0"

	view, err := signedEnvelopeRuntimeView(envelope)
	if err != nil {
		return err
	}
	if view.payloadSchemaID != externalAnchorEvidenceSchemaID || view.payloadSchemaVersion != externalAnchorEvidenceSchemaVersion {
		return fmt.Errorf("payload must be %s@%s", externalAnchorEvidenceSchemaID, externalAnchorEvidenceSchemaVersion)
	}

	payload := view.payload
	canonicalTargetDigestIdentity, proofDigest, err := requireExternalAnchorEvidenceIdentityBindings(payload)
	if err != nil {
		return err
	}
	if err := requireExternalAnchorEvidenceReceiptBindings(payload, canonicalTargetDigestIdentity, proofDigest); err != nil {
		return err
	}
	return requireExternalAnchorEvidenceExpectedBindings(payload)
}

func requireExternalAnchorEvidenceIdentityBindings(payload map[string]any) (string, string, error) {
	canonicalTargetDigestIdentity, err := digestIdentityField(payload, "canonical_target_digest")
	if err != nil {
		return "", "", err
	}
	canonicalTargetIdentity, err := stringField(payload, "canonical_target_identity")
	if err != nil {
		return "", "", err
	}
	if canonicalTargetDigestIdentity != canonicalTargetIdentity {
		return "", "", fmt.Errorf("target identity mismatch: canonical_target_identity does not match canonical_target_digest")
	}
	if err := requireExternalAnchorEvidenceOutcome(payload); err != nil {
		return "", "", err
	}
	sidecarDigests, err := externalAnchorSidecarDigestByKind(payload)
	if err != nil {
		return "", "", err
	}
	proofDigest := sidecarDigests["proof_bytes"]
	if proofDigest == "" {
		return "", "", fmt.Errorf("sidecar_refs must include proof_bytes")
	}
	return canonicalTargetDigestIdentity, proofDigest, nil
}

func requireExternalAnchorEvidenceReceiptBindings(payload map[string]any, canonicalTargetDigestIdentity, proofDigest string) error {
	if receiptTargetDigest, ok, err := optionalDigestIdentityField(payload, "receipt_target_descriptor_digest"); err != nil {
		return err
	} else if ok && receiptTargetDigest != canonicalTargetDigestIdentity {
		return fmt.Errorf("target identity mismatch: receipt_target_descriptor_digest does not match canonical_target_digest")
	}
	if receiptProofDigest, ok, err := optionalDigestIdentityField(payload, "receipt_proof_digest"); err != nil {
		return err
	} else if ok && receiptProofDigest != proofDigest {
		return fmt.Errorf("invalid target proof: receipt_proof_digest does not match sidecar proof_bytes digest")
	}
	return nil
}

func requireExternalAnchorEvidenceExpectedBindings(payload map[string]any) error {
	for _, fieldPair := range [][2]string{{"typed_request_hash", "expected_typed_request_hash"}, {"action_request_hash", "expected_action_request_hash"}, {"approval_request_hash", "expected_approval_request_hash"}, {"approval_decision_hash", "expected_approval_decision_hash"}} {
		if err := requireMatchingDigestBindings(payload, fieldPair[0], fieldPair[1]); err != nil {
			return err
		}
	}
	return nil
}

func requireExternalAnchorEvidenceOutcome(payload map[string]any) error {
	outcome, err := stringField(payload, "outcome")
	if err != nil {
		return err
	}
	switch outcome {
	case "completed", "deferred", "unavailable", "invalid", "failed":
		return nil
	default:
		return fmt.Errorf("unsupported external anchor outcome %q", outcome)
	}
}

func externalAnchorSidecarDigestByKind(payload map[string]any) (map[string]string, error) {
	items, err := requiredArrayValue(payload, "sidecar_refs")
	if err != nil {
		return nil, err
	}
	byKind := map[string]string{}
	for index, item := range items {
		entry, err := objectFromFixtureValue(item, fmt.Sprintf("sidecar_refs[%d]", index))
		if err != nil {
			return nil, err
		}
		kind, err := stringField(entry, "evidence_kind")
		if err != nil {
			return nil, err
		}
		digestIdentity, err := digestIdentityField(entry, "digest")
		if err != nil {
			return nil, err
		}
		if _, exists := byKind[kind]; exists {
			return nil, fmt.Errorf("sidecar_refs includes duplicate evidence_kind %q", kind)
		}
		byKind[kind] = digestIdentity
	}
	return byKind, nil
}

func requireMatchingDigestBindings(payload map[string]any, actualKey string, expectedKey string) error {
	actual, actualOK, err := optionalDigestIdentityField(payload, actualKey)
	if err != nil {
		return err
	}
	expected, expectedOK, err := optionalDigestIdentityField(payload, expectedKey)
	if err != nil {
		return err
	}
	if !expectedOK {
		return nil
	}
	if !actualOK || actual != expected {
		return fmt.Errorf("exact-action binding mismatch: %s does not match %s", actualKey, expectedKey)
	}
	return nil
}

func optionalDigestIdentityField(object map[string]any, key string) (string, bool, error) {
	value, ok := object[key]
	if !ok {
		return "", false, nil
	}
	digest, err := objectFromFixtureValue(value, key)
	if err != nil {
		return "", false, err
	}
	identity, err := digestIdentity(digest)
	if err != nil {
		return "", false, err
	}
	return identity, true, nil
}

func requireImportRestoreReceiptByteIdentity(value map[string]any) error {
	kind, err := stringField(value, "audit_receipt_kind")
	if err != nil {
		return err
	}
	if kind != "import" && kind != "restore" {
		return nil
	}
	payload, err := validateImportRestoreReceiptPayloadShape(value, kind)
	if err != nil {
		return err
	}
	return validateImportRestoreImportedSegments(payload)
}

func validateImportRestoreReceiptPayloadShape(value map[string]any, kind string) (map[string]any, error) {
	payloadSchemaID, err := stringField(value, "receipt_payload_schema_id")
	if err != nil {
		return nil, err
	}
	if payloadSchemaID != "runecode.protocol.audit.receipt.import_restore_provenance.v0" {
		return nil, fmt.Errorf("import/restore receipts must use import_restore provenance payload schema")
	}

	payload, err := requiredObjectField(value, "receipt_payload")
	if err != nil {
		return nil, err
	}

	action, err := stringField(payload, "provenance_action")
	if err != nil {
		return nil, err
	}
	if action != kind {
		return nil, fmt.Errorf("provenance_action %q must match audit_receipt_kind %q", action, kind)
	}

	return payload, nil
}

func validateImportRestoreImportedSegments(payload map[string]any) error {
	segments, err := requiredArrayValue(payload, "imported_segments")
	if err != nil {
		return err
	}
	for index, item := range segments {
		if err := validateImportRestoreSegment(item, index); err != nil {
			return err
		}
	}
	return nil
}

func validateImportRestoreSegment(value any, index int) error {
	segment, err := objectFromFixtureValue(value, fmt.Sprintf("imported_segments[%d]", index))
	if err != nil {
		return err
	}
	verified, err := boolField(segment, "byte_identity_verified")
	if err != nil {
		return err
	}
	if !verified {
		return fmt.Errorf("imported_segments[%d].byte_identity_verified must be true", index)
	}
	sourceHash, err := digestIdentityField(segment, "source_segment_file_hash")
	if err != nil {
		return err
	}
	localHash, err := digestIdentityField(segment, "local_segment_file_hash")
	if err != nil {
		return err
	}
	if sourceHash != localHash {
		return fmt.Errorf("imported_segments[%d] source/local segment hashes differ", index)
	}
	return nil
}

func requireSignedEnvelopePayloadSchemaMatch(value map[string]any, manifest manifestFile, bundle compiledBundle) error {
	envelope, err := signedEnvelopeRuntimeView(value)
	if err != nil {
		return err
	}
	if err := requireRuntimePayloadSchemaMatch(envelope); err != nil {
		return err
	}
	return validateRuntimePayloadSchema(envelope, manifest, bundle)
}

type signedEnvelopeRuntimePayloadView struct {
	payloadSchemaID      string
	payloadSchemaVersion string
	payload              map[string]any
	nestedSchemaID       string
	nestedSchemaVersion  string
}

func signedEnvelopeRuntimeView(value map[string]any) (signedEnvelopeRuntimePayloadView, error) {
	payloadSchemaID, err := stringField(value, "payload_schema_id")
	if err != nil {
		return signedEnvelopeRuntimePayloadView{}, err
	}
	payloadSchemaVersion, err := stringField(value, "payload_schema_version")
	if err != nil {
		return signedEnvelopeRuntimePayloadView{}, err
	}
	payload, err := requiredObjectField(value, "payload")
	if err != nil {
		return signedEnvelopeRuntimePayloadView{}, err
	}
	nestedSchemaID, err := stringField(payload, "schema_id")
	if err != nil {
		return signedEnvelopeRuntimePayloadView{}, err
	}
	nestedSchemaVersion, err := stringField(payload, "schema_version")
	if err != nil {
		return signedEnvelopeRuntimePayloadView{}, err
	}
	return signedEnvelopeRuntimePayloadView{
		payloadSchemaID:      payloadSchemaID,
		payloadSchemaVersion: payloadSchemaVersion,
		payload:              payload,
		nestedSchemaID:       nestedSchemaID,
		nestedSchemaVersion:  nestedSchemaVersion,
	}, nil
}

func requireRuntimePayloadSchemaMatch(envelope signedEnvelopeRuntimePayloadView) error {
	if envelope.payloadSchemaID != envelope.nestedSchemaID {
		return fmt.Errorf("payload_schema_id %q does not match payload.schema_id %q", envelope.payloadSchemaID, envelope.nestedSchemaID)
	}
	if envelope.payloadSchemaVersion != envelope.nestedSchemaVersion {
		return fmt.Errorf("payload_schema_version %q does not match payload.schema_version %q", envelope.payloadSchemaVersion, envelope.nestedSchemaVersion)
	}
	return nil
}

func validateRuntimePayloadSchema(envelope signedEnvelopeRuntimePayloadView, manifest manifestFile, bundle compiledBundle) error {
	schemaPath, err := manifestSchemaPathForRuntimeID(manifest, envelope.payloadSchemaID, envelope.payloadSchemaVersion)
	if err != nil {
		return err
	}
	schemaURI, err := schemaURIFromBundlePath(bundle, schemaPath)
	if err != nil {
		return err
	}
	schema, err := bundle.Compiler.Compile(schemaURI)
	if err != nil {
		return fmt.Errorf("compile schema %q for %s@%s: %w", schemaPath, envelope.payloadSchemaID, envelope.payloadSchemaVersion, err)
	}
	if err := schema.Validate(envelope.payload); err != nil {
		return fmt.Errorf("payload failed %s@%s schema validation: %w", envelope.payloadSchemaID, envelope.payloadSchemaVersion, err)
	}
	return nil
}

func schemaURIFromBundlePath(bundle compiledBundle, schemaPath string) (string, error) {
	schemaDoc, ok := bundle.SchemaDocs[schemaPath]
	if !ok {
		return "", fmt.Errorf("schema document %q not loaded in compiled bundle", schemaPath)
	}
	schemaURI, err := stringField(schemaDoc, "$id")
	if err != nil {
		return "", fmt.Errorf("schema document %q: %w", schemaPath, err)
	}
	return schemaURI, nil
}

func manifestSchemaPathForRuntimeID(manifest manifestFile, schemaID string, schemaVersion string) (string, error) {
	for _, entry := range manifest.SchemaFiles {
		if entry.SchemaID == schemaID && entry.SchemaVersion == schemaVersion {
			return entry.Path, nil
		}
	}

	return "", fmt.Errorf("schema_id %q with schema_version %q not found in manifest", schemaID, schemaVersion)
}

func requireUniqueArtifactDigests(value map[string]any, key string) error {
	items, err := requiredArrayValue(value, key)
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for index, item := range items {
		artifact, err := objectFromFixtureValue(item, fmt.Sprintf("%s[%d]", key, index))
		if err != nil {
			return err
		}
		identity, err := digestIdentityField(artifact, "digest")
		if err != nil {
			return err
		}
		if _, ok := seen[identity]; ok {
			return fmt.Errorf("duplicate artifact digest %s", identity)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func requireUniqueToolIdentities(value map[string]any, key string) error {
	items, err := requiredArrayValue(value, key)
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for index, item := range items {
		tool, err := objectFromFixtureValue(item, fmt.Sprintf("%s[%d]", key, index))
		if err != nil {
			return err
		}
		identity, err := toolIdentity(tool)
		if err != nil {
			return err
		}
		if _, ok := seen[identity]; ok {
			return fmt.Errorf("duplicate tool identity %s", identity)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func requireUniqueToolCallIDs(value map[string]any, key string) error {
	items, err := optionalArrayValue(value, key)
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for index, item := range items {
		toolCall, err := objectFromFixtureValue(item, fmt.Sprintf("%s[%d]", key, index))
		if err != nil {
			return err
		}
		id, err := stringField(toolCall, "tool_call_id")
		if err != nil {
			return err
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate tool_call_id %s", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}
