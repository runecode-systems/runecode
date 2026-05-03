package trustpolicy

const (
	mvpAnchorKindLocalUserPresence      = "local_user_presence_signature"
	mvpAnchorWitnessKindLocalPresenceV0 = "local_user_presence_signature_v0"

	auditReceiptKindProviderInvocationAuthorized = "provider_invocation_authorized"
	auditReceiptKindProviderInvocationDenied     = "provider_invocation_denied"
	auditReceiptKindSecretLeaseIssued            = "secret_lease_issued"
	auditReceiptKindSecretLeaseRevoked           = "secret_lease_revoked"
	auditReceiptKindRuntimeSummary               = "runtime_summary"
	auditReceiptKindDegradedPostureSummary       = "degraded_posture_summary"
	auditReceiptKindNegativeCapabilitySummary    = "negative_capability_summary"
	auditReceiptKindEvidenceBundleExport         = "evidence_bundle_export"
	auditReceiptKindEvidenceImport               = "evidence_import"
	auditReceiptKindEvidenceRestore              = "evidence_restore"
	auditReceiptKindRetentionPolicyChanged       = "retention_policy_changed"
	auditReceiptKindArchivalOperation            = "archival_operation"
	auditReceiptKindVerifierConfigurationChanged = "verifier_configuration_changed"
	auditReceiptKindTrustRootUpdated             = "trust_root_updated"
	auditReceiptKindSensitiveEvidenceView        = "sensitive_evidence_view"

	auditReceiptPayloadSchemaProviderInvocationV0 = "runecode.protocol.audit.receipt.provider_invocation.v0"
	auditReceiptPayloadSchemaSecretLeaseV0        = "runecode.protocol.audit.receipt.secret_lease.v0"
	auditReceiptPayloadSchemaRuntimeSummaryV0     = "runecode.protocol.audit.receipt.runtime_summary.v0"
	auditReceiptPayloadSchemaDegradedPostureV0    = "runecode.protocol.audit.receipt.degraded_posture_summary.v0"
	auditReceiptPayloadSchemaNegativeCapabilityV0 = "runecode.protocol.audit.receipt.negative_capability_summary.v0"
	auditReceiptPayloadSchemaMetaAuditActionV0    = "runecode.protocol.audit.receipt.meta_audit_action.v0"

	networkTargetDescriptorSchemaGatewayDestinationV0 = "runecode.protocol.audit.network_target.gateway_destination.v0"

	anchorKindExternalTransparencyLog = "external_transparency_log_v0"
	anchorKindExternalTimestampAuth   = "external_timestamp_authority_v0"
	anchorKindExternalPublicChain     = "external_public_chain_v0"

	anchorTargetKindTransparencyLog = "transparency_log"
	anchorTargetKindTimestampAuth   = "timestamp_authority"
	anchorTargetKindPublicChain     = "public_chain"

	anchorRuntimeAdapterTransparencyLogV0 = "transparency_log_v0"

	anchorDescriptorSchemaTransparencyLogV0    = "runecode.protocol.audit.anchor_target.transparency_log.v0"
	anchorDescriptorSchemaTimestampAuthorityV0 = "runecode.protocol.audit.anchor_target.timestamp_authority.v0"
	anchorDescriptorSchemaPublicChainV0        = "runecode.protocol.audit.anchor_target.public_chain.v0"

	anchorProofKindTransparencyLogReceiptV0 = "transparency_log_receipt_v0"
	anchorProofKindTimestampTokenV0         = "timestamp_token_v0"
	anchorProofKindPublicChainTxReceiptV0   = "public_chain_tx_receipt_v0"

	anchorProofSchemaTransparencyLogReceiptV0 = "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0"
	anchorProofSchemaTimestampTokenV0         = "runecode.protocol.audit.anchor_proof.timestamp_token.v0"
	anchorProofSchemaPublicChainTxReceiptV0   = "runecode.protocol.audit.anchor_proof.public_chain_tx_receipt.v0"

	anchorDescriptorFieldDescriptorSchemaID   = "descriptor_schema_id"
	anchorDescriptorFieldLogID                = "log_id"
	anchorDescriptorFieldLogPublicKeyDigest   = "log_public_key_digest"
	anchorDescriptorFieldEntryEncodingProfile = "entry_encoding_profile"
	anchorDescriptorFieldAuthorityID          = "authority_id"
	anchorDescriptorFieldCertificateDigest    = "certificate_chain_digest"
	anchorDescriptorFieldTimestampProfile     = "timestamp_profile"
	anchorDescriptorFieldChainNamespace       = "chain_namespace"
	anchorDescriptorFieldNetworkID            = "network_id"
	anchorDescriptorFieldSettlementDigest     = "settlement_contract_digest"
	anchorDerivedFieldSubmitEndpointURI       = "submit_endpoint_uri"
	anchorDerivedFieldTSAEndpointURI          = "tsa_endpoint_uri"
	anchorDerivedFieldRPCEndpointURI          = "rpc_endpoint_uri"
)

var supportedAnchorKinds = map[string]struct{}{
	mvpAnchorKindLocalUserPresence:    {},
	anchorKindExternalTransparencyLog: {},
	anchorKindExternalTimestampAuth:   {},
	anchorKindExternalPublicChain:     {},
}
