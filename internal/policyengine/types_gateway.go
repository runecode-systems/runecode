package policyengine

import "github.com/runecode-ai/runecode/internal/trustpolicy"

type DestinationDescriptor struct {
	SchemaID               string `json:"schema_id"`
	SchemaVersion          string `json:"schema_version"`
	DescriptorKind         string `json:"descriptor_kind"`
	CanonicalHost          string `json:"canonical_host"`
	CanonicalPort          *int   `json:"canonical_port,omitempty"`
	CanonicalPathPrefix    string `json:"canonical_path_prefix,omitempty"`
	ProviderOrNamespace    string `json:"provider_or_namespace,omitempty"`
	GitRepositoryIdentity  string `json:"git_repository_identity,omitempty"`
	TLSRequired            bool   `json:"tls_required"`
	PrivateRangeBlocking   string `json:"private_range_blocking"`
	DNSRebindingProtection string `json:"dns_rebinding_protection"`
}

type GatewayScopeRule struct {
	SchemaID                              string                `json:"schema_id"`
	SchemaVersion                         string                `json:"schema_version"`
	ScopeKind                             string                `json:"scope_kind"`
	GatewayRoleKind                       string                `json:"gateway_role_kind,omitempty"`
	EntryID                               string                `json:"entry_id,omitempty"`
	Destination                           DestinationDescriptor `json:"destination"`
	ExternalAnchorTargetDescriptorDigests []trustpolicy.Digest  `json:"external_anchor_target_descriptor_digests,omitempty"`
	PermittedOperations                   []string              `json:"permitted_operations"`
	AllowedEgressDataClasses              []string              `json:"allowed_egress_data_classes"`
	RedirectPosture                       string                `json:"redirect_posture"`
	MaxTimeoutSeconds                     *int                  `json:"max_timeout_seconds,omitempty"`
	AllowCredentials                      *bool                 `json:"allow_credentials,omitempty"`
	MaxResponseBytes                      *int                  `json:"max_response_bytes,omitempty"`
	GitRefUpdatePolicy                    *GitRefPolicySet      `json:"git_ref_update_policy,omitempty"`
	GitTagUpdatePolicy                    *GitRefPolicySet      `json:"git_tag_update_policy,omitempty"`
	GitPRBaseRefPolicy                    *GitRefPolicySet      `json:"git_pull_request_base_ref_policy,omitempty"`
	GitPRHeadNamespacePolicy              *GitRefPolicySet      `json:"git_pull_request_head_namespace_policy,omitempty"`
}

type GitRefPolicySet struct {
	Rules            []GitRefPolicyRule `json:"rules"`
	AllowForcePush   *bool              `json:"allow_force_push,omitempty"`
	AllowRefDeletion *bool              `json:"allow_ref_deletion,omitempty"`
}

type GitRefPolicyRule struct {
	RuleKind string `json:"rule_kind"`
	Ref      string `json:"ref,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
}
