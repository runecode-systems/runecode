package auditd

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type evidenceBundleProfilePolicy struct {
	Name                       string
	IncludeSegments            bool
	IncludeReceipts            bool
	IncludeVerificationReports bool
	IncludeExternalAnchor      bool
	DefaultPosture             string
	SelectiveDisclosure        bool
}

func evidenceBundleExportProfilePolicy(exportProfile string) (evidenceBundleProfilePolicy, error) {
	switch strings.TrimSpace(exportProfile) {
	case "external_relying_party_minimal":
		return evidenceBundleProfilePolicy{Name: "external_relying_party_minimal", IncludeSegments: false, IncludeReceipts: false, IncludeVerificationReports: true, IncludeExternalAnchor: true, DefaultPosture: "digest_metadata_only", SelectiveDisclosure: true}, nil
	case "company_internal_audit":
		return evidenceBundleProfilePolicy{Name: "company_internal_audit", IncludeSegments: false, IncludeReceipts: true, IncludeVerificationReports: true, IncludeExternalAnchor: true, DefaultPosture: "digest_metadata_only", SelectiveDisclosure: true}, nil
	case "incident_response_scope":
		return evidenceBundleProfilePolicy{Name: "incident_response_scope", IncludeSegments: true, IncludeReceipts: true, IncludeVerificationReports: true, IncludeExternalAnchor: true, DefaultPosture: "operator_private", SelectiveDisclosure: true}, nil
	case "operator_private_full":
		return evidenceBundleProfilePolicy{Name: "operator_private_full", IncludeSegments: true, IncludeReceipts: true, IncludeVerificationReports: true, IncludeExternalAnchor: true, DefaultPosture: "operator_private", SelectiveDisclosure: false}, nil
	default:
		return evidenceBundleProfilePolicy{}, fmt.Errorf("unsupported export profile %q", strings.TrimSpace(exportProfile))
	}
}

func evidenceBundleProfileDeclaredRedactions(policy evidenceBundleProfilePolicy) []AuditEvidenceBundleRedaction {
	out := make([]AuditEvidenceBundleRedaction, 0, 4)
	if !policy.IncludeSegments {
		out = append(out, AuditEvidenceBundleRedaction{Path: "segments/*", ReasonCode: "profile_digest_metadata_default"})
	}
	if !policy.IncludeReceipts {
		out = append(out, AuditEvidenceBundleRedaction{Path: "sidecar/receipts/*", ReasonCode: "profile_digest_metadata_default"})
	}
	if !policy.IncludeVerificationReports {
		out = append(out, AuditEvidenceBundleRedaction{Path: "sidecar/verification-reports/*", ReasonCode: "profile_digest_metadata_default"})
	}
	if !policy.IncludeExternalAnchor {
		out = append(out,
			AuditEvidenceBundleRedaction{Path: "sidecar/external-anchor-evidence/*", ReasonCode: "profile_digest_metadata_default"},
			AuditEvidenceBundleRedaction{Path: "sidecar/external-anchor-sidecars/*", ReasonCode: "profile_digest_metadata_default"},
		)
	}
	return out
}

func applyEvidenceBundleRedactions(objects []AuditEvidenceBundleIncludedObject, redactions []AuditEvidenceBundleRedaction) ([]AuditEvidenceBundleIncludedObject, bool) {
	if len(objects) == 0 || len(redactions) == 0 {
		return objects, false
	}
	out := make([]AuditEvidenceBundleIncludedObject, 0, len(objects))
	applied := false
	for i := range objects {
		if shouldRedactEvidenceBundlePath(objects[i].Path, redactions) {
			applied = true
			continue
		}
		out = append(out, objects[i])
	}
	return out, applied
}

func shouldRedactEvidenceBundlePath(objectPath string, redactions []AuditEvidenceBundleRedaction) bool {
	cleanPath := filepath.ToSlash(strings.TrimSpace(objectPath))
	for i := range redactions {
		if evidenceBundlePathRedactionMatches(cleanPath, redactions[i]) {
			return true
		}
	}
	return false
}

func evidenceBundlePathRedactionMatches(cleanPath string, redaction AuditEvidenceBundleRedaction) bool {
	pattern := filepath.ToSlash(strings.TrimSpace(redaction.Path))
	if pattern == "" {
		return false
	}
	if pattern == cleanPath || strings.HasSuffix(pattern, "/") && strings.HasPrefix(cleanPath, pattern) {
		return true
	}
	matched, err := path.Match(pattern, cleanPath)
	return err == nil && matched
}

func resolveEvidenceBundleDisclosurePosture(requested AuditEvidenceBundleDisclosurePosture, policy evidenceBundleProfilePolicy, userRedactionApplied bool, hasDeclaredRedactions bool) AuditEvidenceBundleDisclosurePosture {
	posture := strings.TrimSpace(requested.Posture)
	if posture == "" {
		posture = policy.DefaultPosture
	}
	selective := requested.SelectiveDisclosureApplied || policy.SelectiveDisclosure || userRedactionApplied || hasDeclaredRedactions
	return AuditEvidenceBundleDisclosurePosture{Posture: posture, SelectiveDisclosureApplied: selective}
}

func evidenceBundleSegmentObjectPath(segmentID string) string {
	cleanSegmentID := path.Clean(strings.TrimSpace(segmentID))
	return filepath.ToSlash(filepath.Join(segmentsDirName, cleanSegmentID+".json"))
}

func previousSealDigestIdentity(seal trustpolicy.AuditSegmentSealPayload) string {
	if seal.PreviousSealDigest == nil {
		return ""
	}
	identity, _ := seal.PreviousSealDigest.Identity()
	return identity
}

func (l *Ledger) evidenceBundleVerifierIdentityLocked() AuditEvidenceBundleVerifierIdentity {
	inputs, err := l.loadVerificationContractInputsOnlyLocked()
	if err != nil || len(inputs.verifierRecords) == 0 {
		return AuditEvidenceBundleVerifierIdentity{}
	}
	record := inputs.verifierRecords[0]
	return AuditEvidenceBundleVerifierIdentity{KeyID: record.KeyID, KeyIDValue: record.KeyIDValue, LogicalPurpose: record.LogicalPurpose, LogicalScope: record.LogicalScope}
}

func (l *Ledger) evidenceBundleTrustRootDigestsLocked() []string {
	inputs, err := l.loadVerificationContractInputsOnlyLocked()
	if err != nil {
		return nil
	}
	set := map[string]struct{}{}
	for i := range inputs.verifierRecords {
		identity, err := digestIdentityForVerifierRecord(inputs.verifierRecords[i])
		if err == nil && identity != "" {
			set[identity] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for identity := range set {
		out = append(out, identity)
	}
	sort.Strings(out)
	return out
}

func validateEvidenceBundleManifestRequest(req AuditEvidenceBundleManifestRequest) error {
	if err := validateEvidenceBundleScope(req.Scope); err != nil {
		return err
	}
	policy, err := validateEvidenceBundleExportProfile(req.ExportProfile)
	if err != nil {
		return err
	}
	if err := validateEvidenceBundleToolIdentity(req.CreatedByTool); err != nil {
		return err
	}
	if err := validateEvidenceBundleDisclosurePosture(req.DisclosurePosture, policy); err != nil {
		return err
	}
	return validateEvidenceBundleArtifactDigests(req.Scope.ArtifactDigests)
}

func validateEvidenceBundleScope(scope AuditEvidenceBundleScope) error {
	switch scopeKind := strings.TrimSpace(scope.ScopeKind); scopeKind {
	case "":
		return fmt.Errorf("bundle scope_kind is required")
	case "run":
		if strings.TrimSpace(scope.RunID) == "" {
			return fmt.Errorf("bundle scope.run_id is required when scope_kind=run")
		}
	case "artifact":
		if len(scope.ArtifactDigests) == 0 {
			return fmt.Errorf("bundle scope.artifact_digests is required when scope_kind=artifact")
		}
	case "incident":
		if strings.TrimSpace(scope.IncidentID) == "" {
			return fmt.Errorf("bundle scope.incident_id is required when scope_kind=incident")
		}
	case "auditor_minimal", "operator_private", "external_relying_party":
		return nil
	default:
		return fmt.Errorf("unsupported bundle scope_kind %q", scopeKind)
	}
	return nil
}

func validateEvidenceBundleExportProfile(exportProfile string) (evidenceBundleProfilePolicy, error) {
	trimmed := strings.TrimSpace(exportProfile)
	if trimmed == "" {
		return evidenceBundleProfilePolicy{}, fmt.Errorf("export profile is required")
	}
	switch trimmed {
	case "operator_private_full", "company_internal_audit", "external_relying_party_minimal", "incident_response_scope":
		return evidenceBundleExportProfilePolicy(trimmed)
	default:
		return evidenceBundleProfilePolicy{}, fmt.Errorf("unsupported export profile %q", exportProfile)
	}
}

func validateEvidenceBundleToolIdentity(identity AuditEvidenceBundleToolIdentity) error {
	if strings.TrimSpace(identity.ToolName) == "" {
		return fmt.Errorf("created_by_tool.tool_name is required")
	}
	if strings.TrimSpace(identity.ToolVersion) == "" {
		return fmt.Errorf("created_by_tool.tool_version is required")
	}
	if hash := strings.TrimSpace(identity.ProtocolBundleManifestHash); hash != "" {
		if _, err := digestFromIdentity(hash); err != nil {
			return fmt.Errorf("created_by_tool.protocol_bundle_manifest_hash: %w", err)
		}
	}
	return nil
}

func validateEvidenceBundleControlProvenance(provenance AuditEvidenceBundleControlProvenance) error {
	for _, field := range []struct {
		label string
		value string
	}{{label: "control_plane_provenance.workflow_definition_hash", value: provenance.WorkflowDefinitionHash}, {label: "control_plane_provenance.tool_manifest_digest", value: provenance.ToolManifestDigest}, {label: "control_plane_provenance.prompt_template_digest", value: provenance.PromptTemplateDigest}, {label: "control_plane_provenance.protocol_bundle_manifest_hash", value: provenance.ProtocolBundleHash}, {label: "control_plane_provenance.verifier_implementation_digest", value: provenance.VerifierImplDigest}, {label: "control_plane_provenance.trust_policy_digest", value: provenance.TrustPolicyDigest}} {
		if digest := strings.TrimSpace(field.value); digest != "" {
			if _, err := digestFromIdentity(digest); err != nil {
				return fmt.Errorf("%s: %w", field.label, err)
			}
		}
	}
	return nil
}

func validateEvidenceBundleDisclosurePosture(posture AuditEvidenceBundleDisclosurePosture, policy evidenceBundleProfilePolicy) error {
	resolved := strings.TrimSpace(posture.Posture)
	if resolved == "" {
		return fmt.Errorf("disclosure posture is required")
	}
	if resolved != policy.DefaultPosture {
		return fmt.Errorf("disclosure_posture.posture %q does not match export profile %q", resolved, policy.Name)
	}
	return nil
}

func validateEvidenceBundleArtifactDigests(artifactDigests []string) error {
	for i := range artifactDigests {
		if _, err := digestFromIdentity(artifactDigests[i]); err != nil {
			return fmt.Errorf("bundle scope.artifact_digests[%d]: %w", i, err)
		}
	}
	return nil
}
