package launcherbackend

type BackendLaunchSpec struct {
	RunID                     string                 `json:"run_id"`
	StageID                   string                 `json:"stage_id"`
	RoleInstanceID            string                 `json:"role_instance_id"`
	RoleFamily                string                 `json:"role_family"`
	RoleKind                  string                 `json:"role_kind"`
	RequestedBackend          string                 `json:"requested_backend"`
	RequestedAccelerationKind string                 `json:"requested_acceleration_kind,omitempty"`
	ControlTransportKind      string                 `json:"control_transport_kind,omitempty"`
	Image                     RuntimeImageDescriptor `json:"runtime_image"`
	Attachments               AttachmentPlan         `json:"attachments"`
	ResourceLimits            BackendResourceLimits  `json:"resource_limits"`
	WatchdogPolicy            BackendWatchdogPolicy  `json:"watchdog_policy"`
	LifecyclePolicy           BackendLifecyclePolicy `json:"lifecycle_policy"`
	CachePosture              BackendCachePosture    `json:"cache_posture"`
}

type BackendResourceLimits struct {
	VCPUCount               int `json:"vcpu_count"`
	MemoryMiB               int `json:"memory_mib"`
	DiskMiB                 int `json:"disk_mib"`
	LaunchTimeoutSeconds    int `json:"launch_timeout_seconds"`
	BindTimeoutSeconds      int `json:"bind_timeout_seconds"`
	ActiveTimeoutSeconds    int `json:"active_timeout_seconds"`
	TerminationGraceSeconds int `json:"termination_grace_seconds"`
}

type BackendWatchdogPolicy struct {
	Enabled                  bool   `json:"enabled"`
	TerminateOnMisbehavior   bool   `json:"terminate_on_misbehavior"`
	HeartbeatTimeoutSeconds  int    `json:"heartbeat_timeout_seconds"`
	NoProgressTimeoutSeconds int    `json:"no_progress_timeout_seconds"`
	TerminationReasonCode    string `json:"termination_reason_code,omitempty"`
}

type BackendLifecyclePolicy struct {
	TerminateBetweenSteps bool `json:"terminate_between_steps"`
}

type BackendLifecycleSnapshot struct {
	CurrentState          string `json:"current_state"`
	PreviousState         string `json:"previous_state,omitempty"`
	TerminateBetweenSteps bool   `json:"terminate_between_steps"`
	TransitionCount       int    `json:"transition_count,omitempty"`
}

type BackendCachePosture struct {
	WarmPoolEnabled               bool `json:"warm_pool_enabled,omitempty"`
	BootCacheEnabled              bool `json:"boot_cache_enabled,omitempty"`
	ResetOrDestroyBeforeReuse     bool `json:"reset_or_destroy_before_reuse"`
	ReusePriorSessionIdentityKeys bool `json:"reuse_prior_session_identity_keys,omitempty"`
	DigestPinned                  bool `json:"digest_pinned"`
	SignaturePinned               bool `json:"signature_pinned"`
}

type BackendCacheEvidence struct {
	ImageCacheResult              string   `json:"image_cache_result"`
	BootArtifactCacheResult       string   `json:"boot_artifact_cache_result"`
	ResolvedImageDescriptorDigest string   `json:"resolved_image_descriptor_digest"`
	ResolvedBootComponentDigests  []string `json:"resolved_boot_component_digests,omitempty"`
}

type LaunchContext struct {
	RunID                string   `json:"run_id"`
	StageID              string   `json:"stage_id"`
	RoleInstanceID       string   `json:"role_instance_id"`
	SessionID            string   `json:"session_id"`
	SessionNonce         string   `json:"session_nonce"`
	ActiveManifestHashes []string `json:"active_manifest_hashes,omitempty"`
	PolicyDecisionRefs   []string `json:"policy_decision_refs,omitempty"`
	ApprovedArtifactRefs []string `json:"approved_artifact_refs,omitempty"`
	LaunchContextDigest  string   `json:"launch_context_digest,omitempty"`
}

type SessionTransportRequirements struct {
	MutualAuthenticationRequired bool `json:"mutual_authentication_required"`
	EncryptionRequired           bool `json:"encryption_required"`
	ReplayProtectionRequired     bool `json:"replay_protection_required"`
}

type SessionFramingContract struct {
	FrameFormat              string `json:"frame_format"`
	MaxFrameBytes            int    `json:"max_frame_bytes"`
	MaxHandshakeMessageBytes int    `json:"max_handshake_message_bytes"`
}

type HostHello struct {
	RunID                 string                       `json:"run_id"`
	StageID               string                       `json:"stage_id"`
	RoleInstanceID        string                       `json:"role_instance_id"`
	IsolateID             string                       `json:"isolate_id"`
	SessionID             string                       `json:"session_id"`
	SessionNonce          string                       `json:"session_nonce"`
	LaunchContextDigest   string                       `json:"launch_context_digest"`
	TransportKind         string                       `json:"transport_kind"`
	TransportRequirements SessionTransportRequirements `json:"transport_requirements"`
	Framing               SessionFramingContract       `json:"framing"`
	HostingNodeID         string                       `json:"hosting_node_id,omitempty"`
}

type IsolateSessionKey struct {
	Alg               string `json:"alg"`
	KeyID             string `json:"key_id"`
	KeyIDValue        string `json:"key_id_value"`
	PublicKeyEncoding string `json:"public_key_encoding"`
	PublicKey         string `json:"public_key"`
	KeyOrigin         string `json:"key_origin"`
}

type SessionKeyProof struct {
	Alg        string `json:"alg"`
	KeyID      string `json:"key_id"`
	KeyIDValue string `json:"key_id_value"`
	Signature  string `json:"signature"`
}

type IsolateHello struct {
	RunID                   string            `json:"run_id"`
	IsolateID               string            `json:"isolate_id"`
	SessionID               string            `json:"session_id"`
	SessionNonce            string            `json:"session_nonce"`
	LaunchContextDigest     string            `json:"launch_context_digest"`
	IsolateSessionKey       IsolateSessionKey `json:"isolate_session_key"`
	ProofOfPossession       SessionKeyProof   `json:"proof_of_possession"`
	HandshakeTranscriptHash string            `json:"handshake_transcript_hash"`
}

type SessionReady struct {
	RunID                     string `json:"run_id"`
	IsolateID                 string `json:"isolate_id"`
	SessionID                 string `json:"session_id"`
	SessionNonce              string `json:"session_nonce"`
	ProvisioningMode          string `json:"provisioning_mode"`
	IdentityBindingPosture    string `json:"identity_binding_posture"`
	IsolateKeyIDValue         string `json:"isolate_key_id_value"`
	HandshakeTranscriptHash   string `json:"handshake_transcript_hash"`
	ChannelKeyMode            string `json:"channel_key_mode"`
	MutuallyAuthenticated     bool   `json:"mutually_authenticated"`
	Encrypted                 bool   `json:"encrypted"`
	ProofOfPossessionVerified bool   `json:"proof_of_possession_verified"`
}

type SessionBindingRecord struct {
	RunID                   string `json:"run_id"`
	IsolateID               string `json:"isolate_id"`
	SessionID               string `json:"session_id"`
	SessionNonce            string `json:"session_nonce"`
	IsolateKeyIDValue       string `json:"isolate_key_id_value"`
	HandshakeTranscriptHash string `json:"handshake_transcript_hash"`
	ProvisioningMode        string `json:"provisioning_mode"`
	IdentityBindingPosture  string `json:"identity_binding_posture"`
}

type SecureSessionIdentity struct {
	Algorithm         string `json:"algorithm"`
	KeyID             string `json:"key_id"`
	KeyIDValue        string `json:"key_id_value"`
	KeyOrigin         string `json:"key_origin"`
	PublicKeyEncoding string `json:"public_key_encoding"`
	PublicKeyDigest   string `json:"public_key_digest"`
}

type SecureSessionChannel struct {
	TransportKind             string `json:"transport_kind"`
	ChannelKeyMode            string `json:"channel_key_mode"`
	FrameFormat               string `json:"frame_format"`
	MaxFrameBytes             int    `json:"max_frame_bytes"`
	MaxHandshakeMessageBytes  int    `json:"max_handshake_message_bytes"`
	MutualAuthentication      bool   `json:"mutual_authentication"`
	Encryption                bool   `json:"encryption"`
	ReplayProtection          bool   `json:"replay_protection"`
	ProofOfPossessionVerified bool   `json:"proof_of_possession_verified"`
}

type SecureSessionSummary struct {
	BindingRecord     SessionBindingRecord   `json:"binding_record"`
	Identity          SecureSessionIdentity  `json:"identity"`
	Channel           SecureSessionChannel   `json:"channel"`
	SecurityPosture   SessionSecurityPosture `json:"security_posture"`
	TranscriptBinding string                 `json:"transcript_binding"`
}

type SessionSecurityPosture struct {
	MutuallyAuthenticated     bool     `json:"mutually_authenticated"`
	Encrypted                 bool     `json:"encrypted"`
	ProofOfPossessionVerified bool     `json:"proof_of_possession_verified"`
	ReplayProtected           bool     `json:"replay_protected"`
	FrameFormat               string   `json:"frame_format"`
	MaxFrameBytes             int      `json:"max_frame_bytes"`
	MaxHandshakeMessageBytes  int      `json:"max_handshake_message_bytes"`
	Degraded                  bool     `json:"degraded"`
	DegradedReasons           []string `json:"degraded_reasons,omitempty"`
}
