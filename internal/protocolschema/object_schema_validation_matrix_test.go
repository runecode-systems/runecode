package protocolschema

import "testing"

func TestErrorSchemaRequiresCategoryRetryabilityAndTypedDetails(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/Error.schema.json")

	for _, testCase := range errorCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestPolicyDecisionRequiresHashBindingsAndApprovalPayloads(t *testing.T) {
	schema := mustCompileObjectSchema(t, newCompiledBundle(t, loadManifest(t)), "objects/PolicyDecision.schema.json")

	for _, testCase := range policyDecisionCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestArtifactAndProvenanceSchemasRequireAuditLinkage(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	artifactSchema := mustCompileObjectSchema(t, bundle, "objects/ArtifactReference.schema.json")
	provenanceSchema := mustCompileObjectSchema(t, bundle, "objects/ProvenanceReceipt.schema.json")

	for _, testCase := range artifactReferenceCases() {
		testCase := testCase
		t.Run("artifact/"+testCase.name, func(t *testing.T) {
			err := artifactSchema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}

	for _, testCase := range artifactDataClassCases() {
		testCase := testCase
		t.Run("artifact-class/"+testCase.name, func(t *testing.T) {
			err := artifactSchema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}

	for _, testCase := range provenanceReceiptCases() {
		testCase := testCase
		t.Run("provenance/"+testCase.name, func(t *testing.T) {
			err := provenanceSchema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestAuditSchemasRequireTypedPayloadsAndSignatures(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	runObjectSchemaCases(t, bundle, "objects/AuditEvent.schema.json", "event", auditEventCases())
	runObjectSchemaCases(t, bundle, "objects/AuditEventContractCatalog.schema.json", "event-contract-catalog", auditEventContractCatalogCases())
	runObjectSchemaCases(t, bundle, "objects/AuditReceipt.schema.json", "receipt", auditReceiptCases())
	runObjectSchemaCases(t, bundle, "objects/AuditSegmentSeal.schema.json", "segment-seal", auditSegmentSealCases())
	runObjectSchemaCases(t, bundle, "objects/AuditSegmentFile.schema.json", "segment-file", auditSegmentFileCases())
	runObjectSchemaCases(t, bundle, "objects/AuditVerificationReport.schema.json", "verification-report", auditVerificationReportCases())
}

func runObjectSchemaCases(t *testing.T, bundle compiledBundle, schemaPath string, prefix string, testCases []validationCase) {
	t.Helper()
	schema := mustCompileObjectSchema(t, bundle, schemaPath)
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(prefix+"/"+testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestArtifactPolicySchemaEncodesFlowAndRetentionControls(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	schema := mustCompileObjectSchema(t, bundle, "objects/ArtifactPolicy.schema.json")

	for _, testCase := range artifactPolicyCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestRuntimeImageDescriptorSchemaRequiresDigestAddressedComponents(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	schema := mustCompileObjectSchema(t, bundle, "objects/RuntimeImageDescriptor.schema.json")

	for _, testCase := range runtimeImageDescriptorCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			err := schema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestIsolateSessionAuditPayloadSchemasValidateReferenceHeavyTopologyNeutralShape(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	startedSchema := mustCompileObjectSchema(t, bundle, "objects/IsolateSessionStartedPayload.schema.json")
	boundSchema := mustCompileObjectSchema(t, bundle, "objects/IsolateSessionBoundPayload.schema.json")

	for _, testCase := range isolateSessionStartedPayloadCases() {
		testCase := testCase
		t.Run("isolate-session-started/"+testCase.name, func(t *testing.T) {
			err := startedSchema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}

	for _, testCase := range isolateSessionBoundPayloadCases() {
		testCase := testCase
		t.Run("isolate-session-bound/"+testCase.name, func(t *testing.T) {
			err := boundSchema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func TestPolicyRuleSetAndAllowlistSchemasValidateDeclarativePolicyInputs(t *testing.T) {
	bundle := newCompiledBundle(t, loadManifest(t))
	ruleSetSchema := mustCompileObjectSchema(t, bundle, "objects/PolicyRuleSet.schema.json")
	allowlistSchema := mustCompileObjectSchema(t, bundle, "objects/PolicyAllowlist.schema.json")
	gatewayScopeRuleSchema := mustCompileObjectSchema(t, bundle, "objects/GatewayScopeRule.schema.json")
	destinationDescriptorSchema := mustCompileObjectSchema(t, bundle, "objects/DestinationDescriptor.schema.json")

	for _, testCase := range policyRuleSetCases() {
		testCase := testCase
		t.Run("rule-set/"+testCase.name, func(t *testing.T) {
			err := ruleSetSchema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}

	for _, testCase := range policyAllowlistCases() {
		testCase := testCase
		t.Run("allowlist/"+testCase.name, func(t *testing.T) {
			err := allowlistSchema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}

	for _, testCase := range gatewayScopeRuleCases() {
		testCase := testCase
		t.Run("gateway-scope-rule/"+testCase.name, func(t *testing.T) {
			err := gatewayScopeRuleSchema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}

	for _, testCase := range destinationDescriptorCases() {
		testCase := testCase
		t.Run("destination-descriptor/"+testCase.name, func(t *testing.T) {
			err := destinationDescriptorSchema.Validate(testCase.value)
			assertValidationOutcome(t, err, testCase.wantErr)
		})
	}
}

func errorCases() []validationCase {
	return []validationCase{
		{name: "minimal error", value: validErrorEnvelope()},
		{name: "typed details pair stays valid", value: validErrorEnvelopeWithDetails()},
		{name: "details require schema id", value: invalidErrorEnvelopeWithoutDetailsSchema(), wantErr: true},
		{name: "error code enforces identifier format", value: invalidErrorEnvelopeCode(), wantErr: true},
		{name: "category enum fails closed", value: invalidErrorEnvelopeCategory(), wantErr: true},
	}
}

func policyDecisionCases() []validationCase {
	return []validationCase{
		{name: "allow decision", value: validAllowPolicyDecision()},
		{name: "deny decision", value: validDenyPolicyDecision()},
		{name: "approval decision", value: validApprovalPolicyDecision()},
		{name: "policy reason code enforces identifier format", value: invalidPolicyDecisionWithBadReasonCode(), wantErr: true},
		{name: "policy reason code must be from registry", value: invalidPolicyDecisionWithUnknownReasonCode(), wantErr: true},
		{name: "approval trigger code must be from registry", value: invalidApprovalPolicyDecisionWithUnknownTriggerCode(), wantErr: true},
		{name: "approval outcome requires payload", value: invalidApprovalPolicyDecisionWithoutPayload(), wantErr: true},
		{name: "deny decision rejects approval payload", value: invalidDenyPolicyDecisionWithApprovalPayload(), wantErr: true},
	}
}

func artifactReferenceCases() []validationCase {
	return []validationCase{
		{name: "artifact reference", value: validArtifactReference()},
		{name: "artifact enforces content type format", value: invalidArtifactReferenceWithBadContentType(), wantErr: true},
		{name: "artifact enforces data class taxonomy", value: invalidArtifactReferenceWithBadDataClass(), wantErr: true},
		{name: "artifact requires provenance", value: invalidArtifactReferenceWithoutProvenance(), wantErr: true},
	}
}

func artifactDataClassCases() []validationCase {
	return []validationCase{
		{name: "spec text class", value: artifactReferenceWithDataClass("spec_text")},
		{name: "unapproved file excerpts class", value: artifactReferenceWithDataClass("unapproved_file_excerpts")},
		{name: "approved file excerpts class", value: artifactReferenceWithDataClass("approved_file_excerpts")},
		{name: "diffs class", value: artifactReferenceWithDataClass("diffs")},
		{name: "build logs class", value: artifactReferenceWithDataClass("build_logs")},
		{name: "audit events class", value: artifactReferenceWithDataClass("audit_events")},
		{name: "audit verification report class", value: artifactReferenceWithDataClass("audit_verification_report")},
		{name: "reserved web query class", value: artifactReferenceWithDataClass("web_query")},
		{name: "reserved web citations class", value: artifactReferenceWithDataClass("web_citations")},
	}
}

func artifactPolicyCases() []validationCase {
	return []validationCase{
		{name: "minimal valid policy", value: validArtifactPolicy()},
		{name: "policy requires hash only handoff mode", value: invalidArtifactPolicyWithNonHashHandoff(), wantErr: true},
		{name: "policy requires explicit approval for promotions", value: invalidArtifactPolicyWithoutExplicitHumanApproval(), wantErr: true},
		{name: "policy rejects unknown flow data class", value: invalidArtifactPolicyWithUnknownFlowDataClass(), wantErr: true},
	}
}

func runtimeImageDescriptorCases() []validationCase {
	return []validationCase{
		{name: "valid descriptor", value: validRuntimeImageDescriptor()},
		{name: "unknown backend kind fails closed", value: invalidRuntimeImageDescriptorWithUnknownBackend(), wantErr: true},
		{name: "platform compatibility required", value: invalidRuntimeImageDescriptorWithoutPlatformCompatibility(), wantErr: true},
		{name: "component digests required", value: invalidRuntimeImageDescriptorWithoutComponents(), wantErr: true},
		{name: "microvm requires kernel and rootfs", value: invalidRuntimeImageDescriptorWithMissingMicroVMKernelRootfs(), wantErr: true},
		{name: "component digest must be digest identity", value: invalidRuntimeImageDescriptorWithBadComponentDigest(), wantErr: true},
		{name: "signing hook object must not be empty", value: invalidRuntimeImageDescriptorWithEmptySigningObject(), wantErr: true},
		{name: "attestation hook digest must be digest identity", value: invalidRuntimeImageDescriptorWithBadAttestationDigest(), wantErr: true},
	}
}

func isolateSessionStartedPayloadCases() []validationCase {
	return []validationCase{
		{name: "valid started payload", value: validIsolateSessionStartedPayload()},
		{name: "invalid started payload schema id", value: invalidIsolateSessionStartedPayloadWithBadSchemaID(), wantErr: true},
		{name: "invalid started payload backend kind", value: invalidIsolateSessionStartedPayloadWithBadBackendKind(), wantErr: true},
		{name: "invalid started payload digest", value: invalidIsolateSessionStartedPayloadWithBadDigest(), wantErr: true},
	}
}

func isolateSessionBoundPayloadCases() []validationCase {
	return []validationCase{
		{name: "valid bound payload", value: validIsolateSessionBoundPayload()},
		{name: "invalid bound payload schema id", value: invalidIsolateSessionBoundPayloadWithBadSchemaID(), wantErr: true},
		{name: "invalid bound payload posture", value: invalidIsolateSessionBoundPayloadWithBadProvisioningPosture(), wantErr: true},
		{name: "invalid bound payload digest", value: invalidIsolateSessionBoundPayloadWithBadDigest(), wantErr: true},
	}
}

func policyRuleSetCases() []validationCase {
	return []validationCase{
		{name: "valid declarative rule set", value: validPolicyRuleSet()},
		{name: "unknown effect fails closed", value: invalidPolicyRuleSetWithUnknownEffect(), wantErr: true},
	}
}

func policyAllowlistCases() []validationCase {
	return []validationCase{
		{name: "valid policy allowlist", value: validPolicyAllowlist()},
		{name: "invalid allowlist kind format", value: invalidPolicyAllowlistKind(), wantErr: true},
		{name: "invalid allowlist entry schema id", value: invalidPolicyAllowlistEntrySchemaID(), wantErr: true},
	}
}

func gatewayScopeRuleCases() []validationCase {
	return []validationCase{
		{name: "valid gateway scope rule", value: validGatewayScopeRule("provider-a")},
		{name: "unknown scope kind fails closed", value: invalidGatewayScopeRuleKind(), wantErr: true},
	}
}

func destinationDescriptorCases() []validationCase {
	return []validationCase{
		{name: "valid destination descriptor", value: validDestinationDescriptor("provider-a")},
		{name: "unknown descriptor kind fails closed", value: invalidDestinationDescriptorKind(), wantErr: true},
	}
}

func provenanceReceiptCases() []validationCase {
	return []validationCase{
		{name: "audit event linkage", value: validProvenanceReceipt()},
		{name: "audit receipt linkage", value: validProvenanceReceiptWithReceiptHash()},
		{name: "audit linkage is mutually exclusive", value: invalidProvenanceReceiptWithBothAuditLinks(), wantErr: true},
		{name: "must link to audit evidence", value: invalidProvenanceReceiptWithoutAuditLinkage(), wantErr: true},
	}
}

func auditEventCases() []validationCase {
	return []validationCase{
		{name: "typed audit event", value: validAuditEvent()},
		{name: "gateway audit event", value: validGatewayAuditEvent()},
		{name: "audit event type enforces identifier format", value: invalidAuditEventWithBadType(), wantErr: true},
		{name: "audit event requires payload hash", value: invalidAuditEventWithoutPayloadHash(), wantErr: true},
		{name: "audit event requires emitter stream id", value: invalidAuditEventWithoutEmitterStreamID(), wantErr: true},
		{name: "audit event requires protocol bundle manifest hash", value: invalidAuditEventWithoutProtocolBundleManifestHash(), wantErr: true},
		{name: "audit event rejects legacy schema bundle hash field", value: invalidAuditEventWithLegacySchemaBundleHash(), wantErr: true},
	}
}

func auditEventContractCatalogCases() []validationCase {
	return []validationCase{
		{name: "valid catalog", value: validAuditEventContractCatalog()},
		{name: "catalog requires entries", value: invalidAuditEventContractCatalogWithoutEntries(), wantErr: true},
		{name: "gateway-required entry needs category", value: invalidAuditEventContractCatalogGatewayRule(), wantErr: true},
	}
}

func auditReceiptCases() []validationCase {
	return []validationCase{
		{name: "minimal receipt", value: validAuditReceipt()},
		{name: "typed payload receipt", value: validAuditReceiptWithPayload()},
		{name: "import provenance receipt", value: validImportAuditReceipt()},
		{name: "restore provenance receipt", value: validRestoreAuditReceipt()},
		{name: "receipt kind enforces identifier format", value: invalidAuditReceiptWithBadKind(), wantErr: true},
		{name: "payload requires schema id", value: invalidAuditReceiptWithoutPayloadSchema(), wantErr: true},
		{name: "import receipt requires import provenance payload", value: invalidImportAuditReceiptWithWrongPayloadSchema(), wantErr: true},
		{name: "import payload requires byte identity verification", value: invalidImportAuditReceiptWithoutByteIdentity(), wantErr: true},
		{name: "restore receipt provenance action must match kind", value: invalidRestoreAuditReceiptWithImportAction(), wantErr: true},
	}
}

func auditSegmentSealCases() []validationCase {
	return []validationCase{
		{name: "minimal segment seal", value: validAuditSegmentSeal()},
		{name: "segment seal supports previous linkage", value: validAuditSegmentSealWithPreviousSeal()},
		{name: "segment seal forbids per run ownership", value: invalidAuditSegmentSealWithPerRunCutOwnership(), wantErr: true},
		{name: "segment seal requires previous digest for non-genesis chain index", value: invalidAuditSegmentSealWithoutPreviousAtNonGenesisIndex(), wantErr: true},
		{name: "segment seal requires event count", value: invalidAuditSegmentSealWithoutEventCount(), wantErr: true},
	}
}

func auditVerificationReportCases() []validationCase {
	return []validationCase{
		{name: "minimal verification report", value: validAuditVerificationReport()},
		{name: "report finding includes digest refs", value: validAuditVerificationReportWithDigestFinding()},
		{name: "invalid finding severity fails", value: invalidAuditVerificationReportWithBadSeverity(), wantErr: true},
	}
}

func auditSegmentFileCases() []validationCase {
	return []validationCase{
		{name: "open segment allows torn trailing marker", value: validOpenAuditSegmentFileWithTornTrailingFrame()},
		{name: "anchored segment is immutable", value: validAnchoredAuditSegmentFile()},
		{name: "imported segment is immutable", value: validImportedAuditSegmentFile()},
		{name: "sealed segment forbids torn trailing marker", value: invalidSealedAuditSegmentFileWithTrailingBytes(), wantErr: true},
		{name: "frame requires digest", value: invalidAuditSegmentFileWithoutFrameDigest(), wantErr: true},
	}
}
