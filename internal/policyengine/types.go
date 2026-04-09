package policyengine

import (
	"encoding/json"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	roleManifestSchemaPath        = "objects/RoleManifest.schema.json"
	runCapabilitySchemaPath       = "objects/CapabilityManifest.schema.json"
	stageCapabilitySchemaPath     = "objects/CapabilityManifest.schema.json"
	allowlistSchemaPath           = "objects/PolicyAllowlist.schema.json"
	destinationSchemaPath         = "objects/DestinationDescriptor.schema.json"
	gatewayScopeRuleSchemaPath    = "objects/GatewayScopeRule.schema.json"
	ruleSetSchemaPath             = "objects/PolicyRuleSet.schema.json"
	actionRequestSchemaPath       = "objects/ActionRequest.schema.json"
	actionKindRegistryPath        = "registries/action_kind.registry.json"
	actionPayloadRegistryPath     = "registries/action_payload_schema_id.registry.json"
	actionPayloadWorkspacePath    = "objects/ActionPayloadWorkspaceWrite.schema.json"
	actionPayloadExecutorPath     = "objects/ActionPayloadExecutorRun.schema.json"
	actionPayloadArtifactPath     = "objects/ActionPayloadArtifactRead.schema.json"
	actionPayloadPromotionPath    = "objects/ActionPayloadPromotion.schema.json"
	actionPayloadGatewayPath      = "objects/ActionPayloadGatewayEgress.schema.json"
	actionPayloadBackendPath      = "objects/ActionPayloadBackendPostureChange.schema.json"
	actionPayloadGatePath         = "objects/ActionPayloadGateOverride.schema.json"
	actionPayloadStageSignPath    = "objects/ActionPayloadStageSummarySignOff.schema.json"
	actionPayloadSecretAccessPath = "objects/ActionPayloadSecretAccess.schema.json"
	policyReasonRegistryPath      = "registries/policy_reason_code.registry.json"
	approvalTriggerRegistryPath   = "registries/approval_trigger_code.registry.json"
	roleManifestSchemaID          = "runecode.protocol.v0.RoleManifest"
	roleManifestSchemaVersion     = "0.2.0"
	capabilityManifestSchemaID    = "runecode.protocol.v0.CapabilityManifest"
	capabilityManifestVersion     = "0.2.0"
	policyAllowlistSchemaID       = "runecode.protocol.v0.PolicyAllowlist"
	policyAllowlistSchemaVersion  = "0.1.0"
	destinationDescriptorSchemaID = "runecode.protocol.v0.DestinationDescriptor"
	destinationDescriptorVersion  = "0.1.0"
	gatewayScopeRuleSchemaID      = "runecode.protocol.v0.GatewayScopeRule"
	gatewayScopeRuleVersion       = "0.1.0"
	policyRuleSetSchemaID         = "runecode.protocol.v0.PolicyRuleSet"
	policyRuleSetSchemaVersion    = "0.1.0"
	actionRequestSchemaID         = "runecode.protocol.v0.ActionRequest"
	actionRequestSchemaVersion    = "0.1.0"

	actionPayloadWorkspaceSchemaID = "runecode.protocol.v0.ActionPayloadWorkspaceWrite"
	actionPayloadExecutorSchemaID  = "runecode.protocol.v0.ActionPayloadExecutorRun"
	actionPayloadArtifactSchemaID  = "runecode.protocol.v0.ActionPayloadArtifactRead"
	actionPayloadPromotionSchemaID = "runecode.protocol.v0.ActionPayloadPromotion"
	actionPayloadGatewaySchemaID   = "runecode.protocol.v0.ActionPayloadGatewayEgress"
	actionPayloadBackendSchemaID   = "runecode.protocol.v0.ActionPayloadBackendPostureChange"
	actionPayloadGateSchemaID      = "runecode.protocol.v0.ActionPayloadGateOverride"
	actionPayloadStageSchemaID     = "runecode.protocol.v0.ActionPayloadStageSummarySignOff"
	actionPayloadSecretAccessID    = "runecode.protocol.v0.ActionPayloadSecretAccess"
)

const (
	ActionKindWorkspaceWrite   = "workspace_write"
	ActionKindExecutorRun      = "executor_run"
	ActionKindArtifactRead     = "artifact_read"
	ActionKindPromotion        = "promotion"
	ActionKindGatewayEgress    = "gateway_egress"
	ActionKindDependencyFetch  = "dependency_fetch"
	ActionKindBackendPosture   = "backend_posture_change"
	ActionKindGateOverride     = "action_gate_override"
	ActionKindStageSummarySign = "stage_summary_sign_off"
	ActionKindSecretAccess     = "secret_access"
)

var actionPayloadByKind = map[string]actionPayloadDescriptor{
	ActionKindWorkspaceWrite:   {schemaID: actionPayloadWorkspaceSchemaID, schemaPath: actionPayloadWorkspacePath},
	ActionKindExecutorRun:      {schemaID: actionPayloadExecutorSchemaID, schemaPath: actionPayloadExecutorPath},
	ActionKindArtifactRead:     {schemaID: actionPayloadArtifactSchemaID, schemaPath: actionPayloadArtifactPath},
	ActionKindPromotion:        {schemaID: actionPayloadPromotionSchemaID, schemaPath: actionPayloadPromotionPath},
	ActionKindGatewayEgress:    {schemaID: actionPayloadGatewaySchemaID, schemaPath: actionPayloadGatewayPath},
	ActionKindDependencyFetch:  {schemaID: actionPayloadGatewaySchemaID, schemaPath: actionPayloadGatewayPath},
	ActionKindBackendPosture:   {schemaID: actionPayloadBackendSchemaID, schemaPath: actionPayloadBackendPath},
	ActionKindGateOverride:     {schemaID: actionPayloadGateSchemaID, schemaPath: actionPayloadGatePath},
	ActionKindStageSummarySign: {schemaID: actionPayloadStageSchemaID, schemaPath: actionPayloadStageSignPath},
	ActionKindSecretAccess:     {schemaID: actionPayloadSecretAccessID, schemaPath: actionPayloadSecretAccessPath},
}

type actionPayloadDescriptor struct {
	schemaID   string
	schemaPath string
}

type ManifestInput struct {
	Payload      json.RawMessage
	ExpectedHash string
}

type CompileInput struct {
	FixedInvariants            FixedInvariants
	RoleManifest               ManifestInput
	RunManifest                ManifestInput
	StageManifest              *ManifestInput
	Allowlists                 []ManifestInput
	RuleSet                    *ManifestInput
	VerifierRecords            []trustpolicy.VerifierRecord
	RequireSignedContextVerify bool
}

type FixedInvariants struct {
	DeniedCapabilities []string
	DeniedActionKinds  []string
}

type ApprovalProfile string

const (
	ApprovalProfileModerate ApprovalProfile = "moderate"
)

type HardFloorOperationClass string

const (
	HardFloorTrustRootChange                  HardFloorOperationClass = "trust_root_change"
	HardFloorSecurityPostureWeakening         HardFloorOperationClass = "security_posture_weakening"
	HardFloorAuthoritativeStateReconciliation HardFloorOperationClass = "authoritative_state_reconciliation"
	HardFloorDeploymentBootstrapAuthority     HardFloorOperationClass = "deployment_bootstrap_authority_change"
)

type ApprovalAssuranceLevel string

const (
	ApprovalAssuranceNone                 ApprovalAssuranceLevel = "none"
	ApprovalAssuranceSessionAuthenticated ApprovalAssuranceLevel = "session_authenticated"
	ApprovalAssuranceReauthenticated      ApprovalAssuranceLevel = "reauthenticated"
	ApprovalAssuranceHardwareBacked       ApprovalAssuranceLevel = "hardware_backed"
)

type RoleManifest struct {
	SchemaID         string               `json:"schema_id"`
	SchemaVersion    string               `json:"schema_version"`
	RoleFamily       string               `json:"role_family"`
	RoleKind         string               `json:"role_kind"`
	ApprovalProfile  string               `json:"approval_profile"`
	CapabilityOptIns []string             `json:"capability_opt_ins"`
	AllowlistRefs    []trustpolicy.Digest `json:"allowlist_refs"`
}

type CapabilityManifest struct {
	SchemaID         string               `json:"schema_id"`
	SchemaVersion    string               `json:"schema_version"`
	ManifestScope    string               `json:"manifest_scope"`
	RunID            string               `json:"run_id,omitempty"`
	StageID          string               `json:"stage_id,omitempty"`
	ApprovalProfile  string               `json:"approval_profile"`
	CapabilityOptIns []string             `json:"capability_opt_ins"`
	AllowlistRefs    []trustpolicy.Digest `json:"allowlist_refs"`
}

type PolicyAllowlist struct {
	SchemaID      string             `json:"schema_id"`
	SchemaVersion string             `json:"schema_version"`
	AllowlistKind string             `json:"allowlist_kind"`
	EntrySchemaID string             `json:"entry_schema_id"`
	Entries       []GatewayScopeRule `json:"entries"`
}

type DestinationDescriptor struct {
	SchemaID               string `json:"schema_id"`
	SchemaVersion          string `json:"schema_version"`
	DescriptorKind         string `json:"descriptor_kind"`
	CanonicalHost          string `json:"canonical_host"`
	CanonicalPort          *int   `json:"canonical_port,omitempty"`
	CanonicalPathPrefix    string `json:"canonical_path_prefix,omitempty"`
	ProviderOrNamespace    string `json:"provider_or_namespace,omitempty"`
	TLSRequired            bool   `json:"tls_required"`
	PrivateRangeBlocking   string `json:"private_range_blocking"`
	DNSRebindingProtection string `json:"dns_rebinding_protection"`
}

type GatewayScopeRule struct {
	SchemaID                 string                `json:"schema_id"`
	SchemaVersion            string                `json:"schema_version"`
	ScopeKind                string                `json:"scope_kind"`
	GatewayRoleKind          string                `json:"gateway_role_kind,omitempty"`
	Destination              DestinationDescriptor `json:"destination"`
	PermittedOperations      []string              `json:"permitted_operations"`
	AllowedEgressDataClasses []string              `json:"allowed_egress_data_classes"`
	RedirectPosture          string                `json:"redirect_posture"`
	AllowCredentials         *bool                 `json:"allow_credentials,omitempty"`
	MaxResponseBytes         *int                  `json:"max_response_bytes,omitempty"`
}

type PolicyRule struct {
	RuleID          string `json:"rule_id"`
	Effect          string `json:"effect"`
	ActionKind      string `json:"action_kind"`
	CapabilityID    string `json:"capability_id,omitempty"`
	ReasonCode      string `json:"reason_code"`
	DetailsSchemaID string `json:"details_schema_id"`
}

type PolicyRuleSet struct {
	SchemaID      string       `json:"schema_id"`
	SchemaVersion string       `json:"schema_version"`
	Rules         []PolicyRule `json:"rules"`
}

type EffectivePolicyContext struct {
	SchemaID                string              `json:"schema_id"`
	SchemaVersion           string              `json:"schema_version"`
	FixedInvariants         FixedInvariants     `json:"fixed_invariants"`
	ActiveRoleFamily        string              `json:"active_role_family"`
	ActiveRoleKind          string              `json:"active_role_kind"`
	ApprovalProfile         ApprovalProfile     `json:"approval_profile"`
	RoleManifestHash        trustpolicy.Digest  `json:"role_manifest_hash"`
	RunManifestHash         trustpolicy.Digest  `json:"run_manifest_hash"`
	StageManifestHash       *trustpolicy.Digest `json:"stage_manifest_hash,omitempty"`
	RoleManifestSignerIDs   []string            `json:"role_manifest_signer_ids,omitempty"`
	RunManifestSignerIDs    []string            `json:"run_manifest_signer_ids,omitempty"`
	StageManifestSignerIDs  []string            `json:"stage_manifest_signer_ids,omitempty"`
	RoleCapabilities        []string            `json:"role_capabilities"`
	RunCapabilities         []string            `json:"run_capabilities"`
	StageCapabilities       []string            `json:"stage_capabilities,omitempty"`
	EffectiveCapabilities   []string            `json:"effective_capabilities"`
	ActiveAllowlistRefs     []string            `json:"active_allowlist_refs"`
	PolicyInputHashes       []string            `json:"policy_input_hashes"`
	EvaluationRuleSetHash   string              `json:"evaluation_rule_set_hash,omitempty"`
	EvaluationRuleSetSchema string              `json:"evaluation_rule_set_schema_id,omitempty"`
}

type CompiledContext struct {
	Context           EffectivePolicyContext
	ManifestHash      string
	PolicyInputHashes []string
	AllowlistsByHash  map[string]PolicyAllowlist
	RuleSet           *PolicyRuleSet
}

type RequiredApproval struct {
	SchemaID string         `json:"schema_id"`
	Payload  map[string]any `json:"payload"`
}

type ActionRequest struct {
	SchemaID               string               `json:"schema_id"`
	SchemaVersion          string               `json:"schema_version"`
	ActionKind             string               `json:"action_kind"`
	CapabilityID           string               `json:"capability_id"`
	AllowlistRefs          []string             `json:"allowlist_refs,omitempty"`
	RelevantArtifactHashes []trustpolicy.Digest `json:"relevant_artifact_hashes,omitempty"`
	ActionPayloadSchemaID  string               `json:"action_payload_schema_id"`
	ActionPayload          map[string]any       `json:"action_payload"`
	ActorKind              string               `json:"actor_kind,omitempty"`
	RoleFamily             string               `json:"role_family,omitempty"`
	RoleKind               string               `json:"role_kind,omitempty"`
}

type DecisionOutcome string

const (
	DecisionAllow                DecisionOutcome = "allow"
	DecisionDeny                 DecisionOutcome = "deny"
	DecisionRequireHumanApproval DecisionOutcome = "require_human_approval"
)

type PolicyDecision struct {
	SchemaID                 string          `json:"schema_id"`
	SchemaVersion            string          `json:"schema_version"`
	DecisionOutcome          DecisionOutcome `json:"decision_outcome"`
	PolicyReasonCode         string          `json:"policy_reason_code"`
	ManifestHash             string          `json:"manifest_hash"`
	PolicyInputHashes        []string        `json:"policy_input_hashes"`
	ActionRequestHash        string          `json:"action_request_hash"`
	RelevantArtifactHashes   []string        `json:"relevant_artifact_hashes"`
	DetailsSchemaID          string          `json:"details_schema_id"`
	Details                  map[string]any  `json:"details"`
	RequiredApprovalSchemaID string          `json:"required_approval_schema_id,omitempty"`
	RequiredApproval         map[string]any  `json:"required_approval,omitempty"`
}
