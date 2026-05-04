package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func projectAuditEvidenceBundleToolIdentity(identity auditd.AuditEvidenceBundleToolIdentity) (AuditEvidenceBundleToolIdentity, error) {
	var protocolBundleManifestHash *trustpolicy.Digest
	if hash := strings.TrimSpace(identity.ProtocolBundleManifestHash); hash != "" {
		d, err := digestFromIdentity(hash)
		if err != nil {
			return AuditEvidenceBundleToolIdentity{}, err
		}
		protocolBundleManifestHash = &d
	}
	return AuditEvidenceBundleToolIdentity{
		ToolName:                   identity.ToolName,
		ToolVersion:                identity.ToolVersion,
		BuildRevision:              identity.BuildRevision,
		ProtocolBundleManifestHash: protocolBundleManifestHash,
	}, nil
}

func projectAuditEvidenceBundleControlProvenance(control *auditd.AuditEvidenceBundleControlProvenance) (*AuditEvidenceBundleControlProvenance, error) {
	if control == nil {
		return nil, nil
	}
	workflowDefinitionHash, err := optionalDigestFromAuditIdentity(control.WorkflowDefinitionHash)
	if err != nil {
		return nil, err
	}
	toolManifestDigest, err := optionalDigestFromAuditIdentity(control.ToolManifestDigest)
	if err != nil {
		return nil, err
	}
	promptTemplateDigest, err := optionalDigestFromAuditIdentity(control.PromptTemplateDigest)
	if err != nil {
		return nil, err
	}
	protocolBundleHash, err := optionalDigestFromAuditIdentity(control.ProtocolBundleHash)
	if err != nil {
		return nil, err
	}
	verifierImplDigest, err := optionalDigestFromAuditIdentity(control.VerifierImplDigest)
	if err != nil {
		return nil, err
	}
	trustPolicyDigest, err := optionalDigestFromAuditIdentity(control.TrustPolicyDigest)
	if err != nil {
		return nil, err
	}
	if workflowDefinitionHash == nil && toolManifestDigest == nil && promptTemplateDigest == nil && protocolBundleHash == nil && verifierImplDigest == nil && trustPolicyDigest == nil {
		return nil, nil
	}
	return &AuditEvidenceBundleControlProvenance{
		WorkflowDefinitionHash: workflowDefinitionHash,
		ToolManifestDigest:     toolManifestDigest,
		PromptTemplateDigest:   promptTemplateDigest,
		ProtocolBundleHash:     protocolBundleHash,
		VerifierImplDigest:     verifierImplDigest,
		TrustPolicyDigest:      trustPolicyDigest,
	}, nil
}

func projectAuditEvidenceBundleVerifierIdentity(identity auditd.AuditEvidenceBundleVerifierIdentity) AuditEvidenceBundleVerifierIdentity {
	return AuditEvidenceBundleVerifierIdentity{
		KeyID:          identity.KeyID,
		KeyIDValue:     identity.KeyIDValue,
		LogicalPurpose: identity.LogicalPurpose,
		LogicalScope:   identity.LogicalScope,
	}
}

func projectAuditEvidenceBundleDisclosurePosture(posture auditd.AuditEvidenceBundleDisclosurePosture) AuditEvidenceBundleDisclosurePosture {
	return AuditEvidenceBundleDisclosurePosture{Posture: posture.Posture, SelectiveDisclosureApplied: posture.SelectiveDisclosureApplied}
}

func projectAuditEvidenceBundleRedactions(redactions []auditd.AuditEvidenceBundleRedaction) []AuditEvidenceBundleRedaction {
	if len(redactions) == 0 {
		return nil
	}
	out := make([]AuditEvidenceBundleRedaction, 0, len(redactions))
	for i := range redactions {
		out = append(out, AuditEvidenceBundleRedaction{Path: redactions[i].Path, ReasonCode: redactions[i].ReasonCode})
	}
	return out
}

func projectAuditEvidenceBundleToolIdentityToTrusted(identity AuditEvidenceBundleToolIdentity) (auditd.AuditEvidenceBundleToolIdentity, error) {
	protocolBundleManifestHash := ""
	if identity.ProtocolBundleManifestHash != nil {
		id, err := identity.ProtocolBundleManifestHash.Identity()
		if err != nil {
			return auditd.AuditEvidenceBundleToolIdentity{}, err
		}
		protocolBundleManifestHash = id
	}
	return auditd.AuditEvidenceBundleToolIdentity{
		ToolName:                   identity.ToolName,
		ToolVersion:                identity.ToolVersion,
		BuildRevision:              identity.BuildRevision,
		ProtocolBundleManifestHash: protocolBundleManifestHash,
	}, nil
}

func projectAuditEvidenceBundleDisclosurePostureToTrusted(posture AuditEvidenceBundleDisclosurePosture) auditd.AuditEvidenceBundleDisclosurePosture {
	return auditd.AuditEvidenceBundleDisclosurePosture{Posture: posture.Posture, SelectiveDisclosureApplied: posture.SelectiveDisclosureApplied}
}

func projectAuditEvidenceBundleRedactionsToTrusted(redactions []AuditEvidenceBundleRedaction) []auditd.AuditEvidenceBundleRedaction {
	if len(redactions) == 0 {
		return nil
	}
	out := make([]auditd.AuditEvidenceBundleRedaction, 0, len(redactions))
	for i := range redactions {
		out = append(out, auditd.AuditEvidenceBundleRedaction{Path: redactions[i].Path, ReasonCode: redactions[i].ReasonCode})
	}
	return out
}
