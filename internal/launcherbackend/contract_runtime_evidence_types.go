package launcherbackend

type LaunchRuntimeEvidence struct {
	RunID                            string                      `json:"run_id"`
	StageID                          string                      `json:"stage_id"`
	RoleInstanceID                   string                      `json:"role_instance_id"`
	RoleFamily                       string                      `json:"role_family,omitempty"`
	RoleKind                         string                      `json:"role_kind,omitempty"`
	BackendKind                      string                      `json:"backend_kind"`
	IsolationAssuranceLevel          string                      `json:"isolation_assurance_level"`
	ProvisioningPosture              string                      `json:"provisioning_posture,omitempty"`
	IsolateID                        string                      `json:"isolate_id,omitempty"`
	SessionID                        string                      `json:"session_id,omitempty"`
	SessionNonce                     string                      `json:"session_nonce,omitempty"`
	LaunchContextDigest              string                      `json:"launch_context_digest,omitempty"`
	HandshakeTranscriptHash          string                      `json:"handshake_transcript_hash,omitempty"`
	IsolateSessionKeyIDValue         string                      `json:"isolate_session_key_id_value,omitempty"`
	HostingNodeID                    string                      `json:"hosting_node_id,omitempty"`
	TransportKind                    string                      `json:"transport_kind,omitempty"`
	HypervisorImplementation         string                      `json:"hypervisor_implementation,omitempty"`
	AccelerationKind                 string                      `json:"acceleration_kind,omitempty"`
	QEMUProvenance                   *QEMUProvenance             `json:"qemu_provenance,omitempty"`
	RuntimeImageDescriptorDigest     string                      `json:"runtime_image_descriptor_digest,omitempty"`
	RuntimeImageBootProfile          string                      `json:"runtime_image_boot_profile,omitempty"`
	RuntimeImageSignerRef            string                      `json:"runtime_image_signer_ref,omitempty"`
	RuntimeImageVerifierRef          string                      `json:"runtime_image_verifier_ref,omitempty"`
	RuntimeImageSignatureDigest      string                      `json:"runtime_image_signature_digest,omitempty"`
	RuntimeToolchainDescriptorDigest string                      `json:"runtime_toolchain_descriptor_digest,omitempty"`
	RuntimeToolchainSignerRef        string                      `json:"runtime_toolchain_signer_ref,omitempty"`
	RuntimeToolchainVerifierRef      string                      `json:"runtime_toolchain_verifier_ref,omitempty"`
	RuntimeToolchainSignatureDigest  string                      `json:"runtime_toolchain_signature_digest,omitempty"`
	AuthorityStateDigest             string                      `json:"authority_state_digest,omitempty"`
	AuthorityStateRevision           uint64                      `json:"authority_state_revision,omitempty"`
	BootComponentDigestByName        map[string]string           `json:"boot_component_digest_by_name,omitempty"`
	BootComponentDigests             []string                    `json:"boot_component_digests,omitempty"`
	AttachmentPlanSummary            *AttachmentPlanSummary      `json:"attachment_plan_summary,omitempty"`
	WorkspaceEncryptionPosture       *WorkspaceEncryptionPosture `json:"workspace_encryption_posture,omitempty"`
	CachePosture                     *BackendCachePosture        `json:"cache_posture,omitempty"`
	CacheEvidence                    *BackendCacheEvidence       `json:"cache_evidence,omitempty"`
	EvidenceDigest                   string                      `json:"evidence_digest"`
}

type SessionRuntimeEvidence struct {
	RunID                    string                  `json:"run_id"`
	IsolateID                string                  `json:"isolate_id"`
	SessionID                string                  `json:"session_id"`
	SessionNonce             string                  `json:"session_nonce"`
	LaunchContextDigest      string                  `json:"launch_context_digest"`
	HandshakeTranscriptHash  string                  `json:"handshake_transcript_hash"`
	IsolateSessionKeyIDValue string                  `json:"isolate_session_key_id_value"`
	ProvisioningPosture      string                  `json:"provisioning_posture"`
	SessionSecurity          *SessionSecurityPosture `json:"session_security,omitempty"`
	EvidenceDigest           string                  `json:"evidence_digest"`
}

type HardeningRuntimeEvidence struct {
	Posture        AppliedHardeningPosture `json:"posture"`
	EvidenceDigest string                  `json:"evidence_digest"`
}

type TerminalRuntimeEvidence struct {
	Report         BackendTerminalReport `json:"report"`
	EvidenceDigest string                `json:"evidence_digest"`
}

type RuntimeEvidenceSnapshot struct {
	Launch                  LaunchRuntimeEvidence                 `json:"launch"`
	Session                 *SessionRuntimeEvidence               `json:"session,omitempty"`
	Attestation             *IsolateAttestationEvidence           `json:"attestation,omitempty"`
	AttestationVerification *IsolateAttestationVerificationRecord `json:"attestation_verification,omitempty"`
	Hardening               HardeningRuntimeEvidence              `json:"hardening"`
	Terminal                *TerminalRuntimeEvidence              `json:"terminal,omitempty"`
}

type IsolateAttestationEvidence struct {
	RunID                        string   `json:"run_id"`
	IsolateID                    string   `json:"isolate_id"`
	SessionID                    string   `json:"session_id"`
	SessionNonce                 string   `json:"session_nonce"`
	HandshakeTranscriptHash      string   `json:"handshake_transcript_hash"`
	IsolateSessionKeyIDValue     string   `json:"isolate_session_key_id_value"`
	LaunchRuntimeEvidenceDigest  string   `json:"launch_runtime_evidence_digest"`
	RuntimeImageDescriptorDigest string   `json:"runtime_image_descriptor_digest"`
	RuntimeImageBootProfile      string   `json:"runtime_image_boot_profile"`
	BootComponentDigests         []string `json:"boot_component_digests,omitempty"`
	AttestationSourceKind        string   `json:"attestation_source_kind"`
	MeasurementProfile           string   `json:"measurement_profile"`
	FreshnessMaterial            []string `json:"freshness_material,omitempty"`
	FreshnessBindingClaims       []string `json:"freshness_binding_claims,omitempty"`
	EvidenceClaimsDigest         string   `json:"evidence_claims_digest,omitempty"`
	EvidenceDigest               string   `json:"evidence_digest"`
}

type IsolateAttestationVerificationRecord struct {
	AttestationEvidenceDigest       string   `json:"attestation_evidence_digest"`
	ReplayIdentityDigest            string   `json:"replay_identity_digest,omitempty"`
	VerifierPolicyID                string   `json:"verifier_policy_id"`
	VerifierPolicyDigest            string   `json:"verifier_policy_digest"`
	VerificationRulesProfileVersion string   `json:"verification_rules_profile_version,omitempty"`
	VerificationTimestamp           string   `json:"verification_timestamp,omitempty"`
	VerificationResult              string   `json:"verification_result"`
	ReasonCodes                     []string `json:"reason_codes,omitempty"`
	ReplayVerdict                   string   `json:"replay_verdict,omitempty"`
	DerivedMeasurementDigests       []string `json:"derived_measurement_digests,omitempty"`
	VerificationDigest              string   `json:"verification_digest"`
}

type RuntimeLifecycleState struct {
	BackendLifecycle            *BackendLifecycleSnapshot `json:"backend_lifecycle,omitempty"`
	ProvisioningPosture         string                    `json:"provisioning_posture,omitempty"`
	ProvisioningPostureDegraded bool                      `json:"provisioning_posture_degraded,omitempty"`
	ProvisioningDegradedReasons []string                  `json:"provisioning_degraded_reasons,omitempty"`
	LaunchFailureReasonCode     string                    `json:"launch_failure_reason_code,omitempty"`
}

type launchRuntimeEvidenceDigestFields struct {
	RunID                            string                      `json:"run_id"`
	StageID                          string                      `json:"stage_id"`
	RoleInstanceID                   string                      `json:"role_instance_id"`
	RoleFamily                       string                      `json:"role_family,omitempty"`
	RoleKind                         string                      `json:"role_kind,omitempty"`
	BackendKind                      string                      `json:"backend_kind"`
	IsolationAssuranceLevel          string                      `json:"isolation_assurance_level"`
	ProvisioningPosture              string                      `json:"provisioning_posture,omitempty"`
	IsolateID                        string                      `json:"isolate_id,omitempty"`
	SessionID                        string                      `json:"session_id,omitempty"`
	SessionNonce                     string                      `json:"session_nonce,omitempty"`
	LaunchContextDigest              string                      `json:"launch_context_digest,omitempty"`
	HandshakeTranscriptHash          string                      `json:"handshake_transcript_hash,omitempty"`
	IsolateSessionKeyIDValue         string                      `json:"isolate_session_key_id_value,omitempty"`
	HostingNodeID                    string                      `json:"hosting_node_id,omitempty"`
	TransportKind                    string                      `json:"transport_kind,omitempty"`
	HypervisorImplementation         string                      `json:"hypervisor_implementation,omitempty"`
	AccelerationKind                 string                      `json:"acceleration_kind,omitempty"`
	QEMUProvenance                   *QEMUProvenance             `json:"qemu_provenance,omitempty"`
	RuntimeImageDescriptorDigest     string                      `json:"runtime_image_descriptor_digest,omitempty"`
	RuntimeImageBootProfile          string                      `json:"runtime_image_boot_profile,omitempty"`
	RuntimeImageSignerRef            string                      `json:"runtime_image_signer_ref,omitempty"`
	RuntimeImageVerifierRef          string                      `json:"runtime_image_verifier_ref,omitempty"`
	RuntimeImageSignatureDigest      string                      `json:"runtime_image_signature_digest,omitempty"`
	RuntimeToolchainDescriptorDigest string                      `json:"runtime_toolchain_descriptor_digest,omitempty"`
	RuntimeToolchainSignerRef        string                      `json:"runtime_toolchain_signer_ref,omitempty"`
	RuntimeToolchainVerifierRef      string                      `json:"runtime_toolchain_verifier_ref,omitempty"`
	RuntimeToolchainSignatureDigest  string                      `json:"runtime_toolchain_signature_digest,omitempty"`
	AuthorityStateDigest             string                      `json:"authority_state_digest,omitempty"`
	AuthorityStateRevision           uint64                      `json:"authority_state_revision,omitempty"`
	BootComponentDigestByName        map[string]string           `json:"boot_component_digest_by_name,omitempty"`
	BootComponentDigests             []string                    `json:"boot_component_digests,omitempty"`
	AttachmentPlanSummary            *AttachmentPlanSummary      `json:"attachment_plan_summary,omitempty"`
	WorkspaceEncryptionPosture       *WorkspaceEncryptionPosture `json:"workspace_encryption_posture,omitempty"`
	CachePosture                     *BackendCachePosture        `json:"cache_posture,omitempty"`
	CacheEvidence                    *BackendCacheEvidence       `json:"cache_evidence,omitempty"`
}

type sessionRuntimeEvidenceDigestFields struct {
	RunID                    string                  `json:"run_id"`
	IsolateID                string                  `json:"isolate_id"`
	SessionID                string                  `json:"session_id"`
	SessionNonce             string                  `json:"session_nonce"`
	LaunchContextDigest      string                  `json:"launch_context_digest"`
	HandshakeTranscriptHash  string                  `json:"handshake_transcript_hash"`
	IsolateSessionKeyIDValue string                  `json:"isolate_session_key_id_value"`
	ProvisioningPosture      string                  `json:"provisioning_posture"`
	SessionSecurity          *SessionSecurityPosture `json:"session_security,omitempty"`
}

type isolateAttestationEvidenceDigestFields struct {
	RunID                        string   `json:"run_id"`
	IsolateID                    string   `json:"isolate_id"`
	SessionID                    string   `json:"session_id"`
	SessionNonce                 string   `json:"session_nonce"`
	HandshakeTranscriptHash      string   `json:"handshake_transcript_hash"`
	IsolateSessionKeyIDValue     string   `json:"isolate_session_key_id_value"`
	LaunchRuntimeEvidenceDigest  string   `json:"launch_runtime_evidence_digest"`
	RuntimeImageDescriptorDigest string   `json:"runtime_image_descriptor_digest"`
	RuntimeImageBootProfile      string   `json:"runtime_image_boot_profile"`
	BootComponentDigests         []string `json:"boot_component_digests,omitempty"`
	AttestationSourceKind        string   `json:"attestation_source_kind"`
	MeasurementProfile           string   `json:"measurement_profile"`
	FreshnessMaterial            []string `json:"freshness_material,omitempty"`
	FreshnessBindingClaims       []string `json:"freshness_binding_claims,omitempty"`
	EvidenceClaimsDigest         string   `json:"evidence_claims_digest,omitempty"`
}

type isolateAttestationVerificationRecordDigestFields struct {
	AttestationEvidenceDigest       string   `json:"attestation_evidence_digest"`
	ReplayIdentityDigest            string   `json:"replay_identity_digest,omitempty"`
	VerifierPolicyID                string   `json:"verifier_policy_id"`
	VerifierPolicyDigest            string   `json:"verifier_policy_digest"`
	VerificationRulesProfileVersion string   `json:"verification_rules_profile_version,omitempty"`
	VerificationTimestamp           string   `json:"verification_timestamp,omitempty"`
	VerificationResult              string   `json:"verification_result"`
	ReasonCodes                     []string `json:"reason_codes,omitempty"`
	ReplayVerdict                   string   `json:"replay_verdict,omitempty"`
	DerivedMeasurementDigests       []string `json:"derived_measurement_digests,omitempty"`
}

type isolateAttestationReplayIdentityFields struct {
	RunID                     string `json:"run_id"`
	IsolateID                 string `json:"isolate_id"`
	SessionID                 string `json:"session_id"`
	SessionNonce              string `json:"session_nonce"`
	HandshakeTranscriptHash   string `json:"handshake_transcript_hash"`
	IsolateSessionKeyIDValue  string `json:"isolate_session_key_id_value"`
	LaunchEvidenceDigest      string `json:"launch_evidence_digest"`
	AttestationEvidenceDigest string `json:"attestation_evidence_digest"`
	MeasurementProfile        string `json:"measurement_profile"`
}
