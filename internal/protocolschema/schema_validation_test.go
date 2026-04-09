package protocolschema

import (
	"strings"
	"testing"
)

func TestSchemasCompileAgainstDraft202012(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))

	for filePath, schemaDoc := range bundle.SchemaDocs {
		filePath := filePath
		schemaDoc := schemaDoc
		t.Run(filePath, func(t *testing.T) {
			schemaID := stringValue(t, schemaDoc, "$id")
			if _, err := bundle.Compiler.Compile(schemaID); err != nil {
				t.Fatalf("Compile(%q) for %q returned error: %v", schemaID, filePath, err)
			}
		})
	}
}

func TestSchemaPropertiesHaveClassificationBoundsAndDescriptions(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))

	for filePath, schemaDoc := range bundle.SchemaDocs {
		filePath := filePath
		schemaDoc := schemaDoc
		t.Run(filePath, func(t *testing.T) {
			assertSchemaNodeInvariants(t, filePath, schemaDoc, false)
			assertReferencedDefinitions(t, filePath, schemaDoc, bundle.SchemaDocs, map[string]struct{}{})
		})
	}
}

func TestDigestSchemaPinsSHA256(t *testing.T) {
	schema := loadJSONMap(t, schemaPath(t, "objects/Digest.schema.json"))
	required := stringSliceValue(t, schema, "required")
	assertContains(t, required, "hash_alg")
	assertContains(t, required, "hash")

	properties := objectValue(t, schema, "properties")
	assertConst(t, properties, "hash_alg", "sha256")

	digestValue := objectValue(t, objectValue(t, schema, "$defs"), "digestValue")
	digestRequired := stringSliceValue(t, digestValue, "required")
	assertContains(t, digestRequired, "hash_alg")
	assertContains(t, digestRequired, "hash")
}

func TestSignedEnvelopeConstrainsPayloadAndAlgorithms(t *testing.T) {
	schema := loadJSONMap(t, schemaPath(t, "objects/SignedObjectEnvelope.schema.json"))
	required := stringSliceValue(t, schema, "required")
	assertContains(t, required, "payload")
	assertContains(t, required, "signature_input")
	assertContains(t, required, "signature")

	if got := strings.TrimSpace(stringValue(t, schema, "description")); !strings.Contains(got, "single-signature") {
		t.Fatalf("SignedObjectEnvelope description must declare single-signature posture; got: %q", got)
	}

	properties := objectValue(t, schema, "properties")
	assertConst(t, properties, "signature_input", "rfc8785_jcs_detached_payload")
	assertSignedEnvelopePayload(t, objectValue(t, properties, "payload"))
	assertSignatureBlock(t, objectValue(t, objectValue(t, schema, "$defs"), "signatureBlock"))
}

func TestAuditSchemasUseCanonicalAuditRecordDigestReferences(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))

	auditEvent := bundle.SchemaDocs["objects/AuditEvent.schema.json"]
	auditReceipt := bundle.SchemaDocs["objects/AuditReceipt.schema.json"]
	auditSegmentSeal := bundle.SchemaDocs["objects/AuditSegmentSeal.schema.json"]
	auditSegmentFile := bundle.SchemaDocs["objects/AuditSegmentFile.schema.json"]
	auditVerificationReport := bundle.SchemaDocs["objects/AuditVerificationReport.schema.json"]

	assertRefEquals(t, objectValue(t, objectValue(t, auditEvent, "properties"), "previous_event_hash"), "$ref", "AuditRecordDigest.schema.json#/$defs/auditRecordDigestValue")
	assertRefEquals(t, objectValue(t, objectValue(t, auditReceipt, "properties"), "subject_digest"), "$ref", "AuditRecordDigest.schema.json#/$defs/auditRecordDigestValue")
	assertRefEquals(t, objectValue(t, objectValue(t, auditSegmentSeal, "properties"), "first_record_digest"), "$ref", "AuditRecordDigest.schema.json#/$defs/auditRecordDigestValue")
	assertRefEquals(t, objectValue(t, objectValue(t, auditSegmentSeal, "properties"), "last_record_digest"), "$ref", "AuditRecordDigest.schema.json#/$defs/auditRecordDigestValue")
	assertRefEquals(t, objectValue(t, objectValue(t, auditSegmentSeal, "properties"), "previous_seal_digest"), "$ref", "AuditRecordDigest.schema.json#/$defs/auditRecordDigestValue")
	assertConst(t, objectValue(t, auditSegmentSeal, "properties"), "sealed_after_state", "open")
	assertConst(t, objectValue(t, auditSegmentSeal, "properties"), "merkle_profile", "sha256_ordered_dse_v1")
	assertConst(t, objectValue(t, auditSegmentSeal, "properties"), "segment_file_hash_scope", "raw_framed_segment_bytes_v1")
	assertConst(t, objectValue(t, auditSegmentSeal, "properties"), "anchoring_subject", "audit_segment_seal")

	recordFrame := objectValue(t, objectValue(t, auditSegmentFile, "$defs"), "recordFrame")
	assertRefEquals(t, objectValue(t, objectValue(t, recordFrame, "properties"), "record_digest"), "$ref", "AuditRecordDigest.schema.json#/$defs/auditRecordDigestValue")

	finding := objectValue(t, objectValue(t, auditVerificationReport, "$defs"), "finding")
	assertRefEquals(t, objectValue(t, objectValue(t, finding, "properties"), "subject_record_digest"), "$ref", "AuditRecordDigest.schema.json#/$defs/auditRecordDigestValue")
	assertContains(t, stringSliceValue(t, finding, "required"), "dimension")
}

func TestAuditEventSchemaUsesDetachedPayloadContractAndHashBindings(t *testing.T) {
	auditEvent := loadJSONMap(t, schemaPath(t, "objects/AuditEvent.schema.json"))
	properties := objectValue(t, auditEvent, "properties")
	required := stringSliceValue(t, auditEvent, "required")

	assertContains(t, required, "emitter_stream_id")
	assertContains(t, required, "seq")
	assertContains(t, required, "protocol_bundle_manifest_hash")
	assertContains(t, required, "event_payload_hash")

	if hasKey(properties, "signatures") {
		t.Fatal("AuditEvent must not include inline signatures; use SignedObjectEnvelope instead")
	}
	if hasKey(properties, "manifest_hash") {
		t.Fatal("AuditEvent must bind trust context with explicit role/capability hashes, not legacy manifest_hash")
	}
	if hasKey(properties, "schema_bundle_manifest_hash") {
		t.Fatal("AuditEvent must use protocol_bundle_manifest_hash, not legacy schema_bundle_manifest_hash")
	}

	assertRefEquals(t, objectValue(t, properties, "protocol_bundle_manifest_hash"), "$ref", "Digest.schema.json#/$defs/digestValue")
	assertRefEquals(t, objectValue(t, properties, "active_role_manifest_hash"), "$ref", "Digest.schema.json#/$defs/digestValue")
	assertRefEquals(t, objectValue(t, properties, "active_capability_manifest_hash"), "$ref", "Digest.schema.json#/$defs/digestValue")
}

func assertRefEquals(t *testing.T, node map[string]any, key string, want string) {
	t.Helper()
	if got := stringValue(t, node, key); got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}

func TestManifestsRequireExplicitSignedInputs(t *testing.T) {
	for _, schemaFile := range []string{
		"objects/RoleManifest.schema.json",
		"objects/CapabilityManifest.schema.json",
	} {
		schemaFile := schemaFile
		t.Run(schemaFile, func(t *testing.T) {
			schema := loadJSONMap(t, schemaPath(t, schemaFile))
			required := stringSliceValue(t, schema, "required")
			assertContains(t, required, "principal")
			assertContains(t, required, "approval_profile")
			assertContains(t, required, "capability_opt_ins")
			assertContains(t, required, "allowlist_refs")
			assertContains(t, required, "signatures")

			approvalProfile := objectValue(t, objectValue(t, schema, "properties"), "approval_profile")
			assertContains(t, stringSliceValue(t, approvalProfile, "enum"), "moderate")
		})
	}

	capabilitySchema := loadJSONMap(t, schemaPath(t, "objects/CapabilityManifest.schema.json"))
	properties := objectValue(t, capabilitySchema, "properties")
	manifestScope := objectValue(t, properties, "manifest_scope")
	enumValues := stringSliceValue(t, manifestScope, "enum")
	assertContains(t, enumValues, "run")
	assertContains(t, enumValues, "stage")
}

func TestPrincipalIdentityConstrainsRoleKindByActorKind(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/PrincipalIdentity.schema.json")

	for _, testCase := range principalIdentityCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func assertSignedEnvelopePayload(t *testing.T, payload map[string]any) {
	t.Helper()

	if got := stringValue(t, payload, "type"); got != "object" {
		t.Fatalf("payload type = %q, want object", got)
	}
	payloadRequired := stringSliceValue(t, payload, "required")
	assertContains(t, payloadRequired, "schema_id")
	assertContains(t, payloadRequired, "schema_version")
	if got := stringValue(t, payload, "x-data-class"); got != "secret" {
		t.Fatalf("payload x-data-class = %q, want secret", got)
	}
}

func assertSignatureBlock(t *testing.T, signatureBlock map[string]any) {
	t.Helper()

	signatureRequired := stringSliceValue(t, signatureBlock, "required")
	assertContains(t, signatureRequired, "alg")
	assertContains(t, signatureRequired, "key_id")
	assertContains(t, signatureRequired, "key_id_value")
	assertContains(t, signatureRequired, "signature")

	properties := objectValue(t, signatureBlock, "properties")
	alg := objectValue(t, properties, "alg")
	assertContains(t, stringSliceValue(t, alg, "enum"), "ed25519")
	assertConst(t, properties, "key_id", "key_sha256")
	keyIDValue := objectValue(t, properties, "key_id_value")
	if got := stringValue(t, keyIDValue, "pattern"); got != "^[a-f0-9]{64}$" {
		t.Fatalf("key_id_value pattern = %q, want ^[a-f0-9]{64}$", got)
	}
}

func assertSchemaNodeInvariants(t *testing.T, location string, node map[string]any, requireClassification bool) {
	t.Helper()

	assertNodeClassification(t, location, node, requireClassification)
	assertNodeStructuralBounds(t, location, node)
	assertPropertyNodeInvariants(t, location, node)
	assertDefinitionNodeInvariants(t, location, node)
	assertItemNodeInvariants(t, location, node)
}

func assertNodeClassification(t *testing.T, location string, node map[string]any, requireClassification bool) {
	t.Helper()

	if !requireClassification {
		return
	}

	description := strings.TrimSpace(stringValue(t, node, "description"))
	if description == "" {
		t.Fatalf("%s must have a non-empty description", location)
	}

	dataClass := stringValue(t, node, "x-data-class")
	if _, ok := allowedDataClasses[dataClass]; !ok {
		t.Fatalf("%s uses unsupported x-data-class %q", location, dataClass)
	}
}

func assertNodeStructuralBounds(t *testing.T, location string, node map[string]any) {
	t.Helper()

	schemaType, ok := optionalStringValue(node, "type")
	if !ok {
		return
	}

	switch schemaType {
	case "object":
		if !hasNumber(node, "maxProperties") {
			t.Fatalf("%s must declare maxProperties", location)
		}
	case "array":
		if !hasNumber(node, "maxItems") {
			t.Fatalf("%s must declare maxItems", location)
		}
	case "string":
		if !hasNumber(node, "maxLength") && !hasKey(node, "const") && !hasKey(node, "enum") {
			t.Fatalf("%s must declare maxLength or constrain values with const/enum", location)
		}
	}
}

func assertPropertyNodeInvariants(t *testing.T, location string, node map[string]any) {
	t.Helper()

	properties, ok := optionalObjectValue(node, "properties")
	if !ok {
		return
	}

	for _, key := range sortedKeys(properties) {
		child := objectFromAny(t, location+"."+key, properties[key])
		assertSchemaNodeInvariants(t, location+"."+key, child, true)
	}
}

func assertDefinitionNodeInvariants(t *testing.T, location string, node map[string]any) {
	t.Helper()

	defs, ok := optionalObjectValue(node, "$defs")
	if !ok {
		return
	}

	for _, key := range sortedKeys(defs) {
		child := objectFromAny(t, location+".$defs."+key, defs[key])
		if strings.TrimSpace(stringValue(t, child, "description")) == "" {
			t.Fatalf("%s.$defs.%s must have a non-empty description", location, key)
		}
		assertSchemaNodeInvariants(t, location+".$defs."+key, child, false)
	}
}

func assertItemNodeInvariants(t *testing.T, location string, node map[string]any) {
	t.Helper()

	items, ok := optionalObjectValue(node, "items")
	if !ok {
		return
	}

	assertSchemaNodeInvariants(t, location+"[]", items, false)
}

type validationCase struct {
	name    string
	value   map[string]any
	wantErr bool
}

func principalIdentityCases() []validationCase {
	return []validationCase{
		principalIdentityRoleInstanceCase(),
		principalIdentityRoleInstanceMissingRoleKindCase(),
		principalIdentityDaemonCase(),
		principalIdentityExternalRuntimeCase(),
		principalIdentityExternalRuntimeWithRoleKindCase(),
		principalIdentityExternalRuntimeWithRoleKindMissingFamilyCase(),
		principalIdentityExternalRuntimeWorkspaceFamilyWithGatewayKindCase(),
		principalIdentityExternalRuntimeGatewayFamilyWithWorkspaceKindCase(),
		principalIdentityUserWithRoleKindCase(),
		principalIdentityUserWithRoleFamilyCase(),
		principalIdentityLocalClientWithRoleKindCase(),
		principalIdentityLocalClientWithRoleFamilyCase(),
	}
}

func principalIdentityRoleInstanceCase() validationCase {
	return validationCase{
		name: "role instance requires role kind",
		value: map[string]any{
			"schema_id":                       "runecode.protocol.v0.PrincipalIdentity",
			"schema_version":                  "0.2.0",
			"actor_kind":                      "role_instance",
			"principal_id":                    "role-123",
			"instance_id":                     "gateway-1",
			"role_family":                     "gateway",
			"role_kind":                       "model-gateway",
			"active_role_manifest_hash":       testDigestValue("a"),
			"active_capability_manifest_hash": testDigestValue("b"),
		},
	}
}

func principalIdentityRoleInstanceMissingRoleKindCase() validationCase {
	return validationCase{
		name: "role instance without role kind fails",
		value: map[string]any{
			"schema_id":                       "runecode.protocol.v0.PrincipalIdentity",
			"schema_version":                  "0.2.0",
			"actor_kind":                      "role_instance",
			"principal_id":                    "role-123",
			"instance_id":                     "gateway-1",
			"active_role_manifest_hash":       testDigestValue("a"),
			"active_capability_manifest_hash": testDigestValue("b"),
		},
		wantErr: true,
	}
}

func principalIdentityDaemonCase() validationCase {
	return validationCase{
		name: "daemon may include role kind",
		value: map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "daemon",
			"principal_id":   "secretsd",
			"instance_id":    "daemon-1",
			"role_family":    "gateway",
			"role_kind":      "auth-gateway",
		},
	}
}

func principalIdentityExternalRuntimeCase() validationCase {
	return validationCase{
		name: "external runtime may omit role kind",
		value: map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "external_runtime",
			"principal_id":   "provider-runtime",
			"instance_id":    "runtime-1",
		},
	}
}

func principalIdentityExternalRuntimeWithRoleKindCase() validationCase {
	return validationCase{
		name: "external runtime may include role kind",
		value: map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "external_runtime",
			"principal_id":   "provider-runtime",
			"instance_id":    "runtime-1",
			"role_family":    "gateway",
			"role_kind":      "model-gateway",
		},
	}
}

func principalIdentityExternalRuntimeWithRoleKindMissingFamilyCase() validationCase {
	return validationCase{
		name: "role kind requires role family",
		value: map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "external_runtime",
			"principal_id":   "provider-runtime",
			"instance_id":    "runtime-1",
			"role_kind":      "model-gateway",
		},
		wantErr: true,
	}
}

func principalIdentityExternalRuntimeWorkspaceFamilyWithGatewayKindCase() validationCase {
	return validationCase{
		name: "workspace family rejects gateway role kind",
		value: map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "external_runtime",
			"principal_id":   "provider-runtime",
			"instance_id":    "runtime-1",
			"role_family":    "workspace",
			"role_kind":      "model-gateway",
		},
		wantErr: true,
	}
}

func principalIdentityExternalRuntimeGatewayFamilyWithWorkspaceKindCase() validationCase {
	return validationCase{
		name: "gateway family rejects workspace role kind",
		value: map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "external_runtime",
			"principal_id":   "provider-runtime",
			"instance_id":    "runtime-1",
			"role_family":    "gateway",
			"role_kind":      "workspace-edit",
		},
		wantErr: true,
	}
}

func principalIdentityUserWithRoleKindCase() validationCase {
	return validationCase{
		name: "user may not include role kind",
		value: map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "user",
			"principal_id":   "alice",
			"instance_id":    "user-session-1",
			"role_kind":      "model-gateway",
		},
		wantErr: true,
	}
}

func principalIdentityUserWithRoleFamilyCase() validationCase {
	return validationCase{
		name: "user may not include role family",
		value: map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "user",
			"principal_id":   "alice",
			"instance_id":    "user-session-1",
			"role_family":    "workspace",
		},
		wantErr: true,
	}
}

func principalIdentityLocalClientWithRoleKindCase() validationCase {
	return validationCase{
		name: "local client may not include role kind",
		value: map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "local_client",
			"principal_id":   "cli-session",
			"instance_id":    "client-1",
			"role_kind":      "workspace-edit",
		},
		wantErr: true,
	}
}

func principalIdentityLocalClientWithRoleFamilyCase() validationCase {
	return validationCase{
		name: "local client may not include role family",
		value: map[string]any{
			"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
			"schema_version": "0.2.0",
			"actor_kind":     "local_client",
			"principal_id":   "cli-session",
			"instance_id":    "client-1",
			"role_family":    "gateway",
		},
		wantErr: true,
	}
}

func assertValidationOutcome(t *testing.T, err error, wantErr bool) {
	t.Helper()

	if wantErr && err == nil {
		t.Fatal("Validate returned nil error, want failure")
	}
	if !wantErr && err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}
