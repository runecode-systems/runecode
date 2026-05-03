package auditd

import (
	"encoding/json"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) evidenceBundleControlPlaneProvenanceLocked(scope AuditEvidenceBundleScope, segmentIDs []string) (*AuditEvidenceBundleControlProvenance, error) {
	workflow, protocolFromEvent, err := l.evidenceBundleControlPlaneWorkflowAndProtocolLocked(scope, segmentIDs)
	if err != nil {
		return nil, err
	}
	protocol := strings.TrimSpace(protocolFromEvent)
	verifierDigest, trustPolicyDigest, err := l.evidenceBundleVerifierAndTrustPolicyDigestsLocked()
	if err != nil {
		return nil, err
	}
	provenance := normalizeEvidenceBundleControlProvenance(AuditEvidenceBundleControlProvenance{
		WorkflowDefinitionHash: workflow,
		ProtocolBundleHash:     protocol,
		VerifierImplDigest:     verifierDigest,
		TrustPolicyDigest:      trustPolicyDigest,
	})
	if provenance.WorkflowDefinitionHash == "" && provenance.ToolManifestDigest == "" && provenance.PromptTemplateDigest == "" && provenance.ProtocolBundleHash == "" && provenance.VerifierImplDigest == "" && provenance.TrustPolicyDigest == "" {
		return nil, nil
	}
	return &provenance, nil
}

func (l *Ledger) evidenceBundleControlPlaneWorkflowAndProtocolLocked(scope AuditEvidenceBundleScope, segmentIDs []string) (string, string, error) {
	runID := strings.TrimSpace(scope.RunID)
	if runID == "" {
		return "", "", nil
	}
	segmentSet := map[string]struct{}{}
	for i := range segmentIDs {
		segmentID := strings.TrimSpace(segmentIDs[i])
		if segmentID != "" {
			segmentSet[segmentID] = struct{}{}
		}
	}
	workflowCandidates := map[string]struct{}{}
	protocolCandidates := map[string]struct{}{}
	for segmentID := range segmentSet {
		segment, err := l.loadSegment(segmentID)
		if err != nil {
			return "", "", err
		}
		if err := collectControlPlaneDigestsFromSegment(segment, runID, workflowCandidates, protocolCandidates); err != nil {
			return "", "", err
		}
	}
	workflow := firstIdentityFromSet(workflowCandidates)
	protocol := firstIdentityFromSet(protocolCandidates)
	return workflow, protocol, nil
}

func collectControlPlaneDigestsFromSegment(segment trustpolicy.AuditSegmentFilePayload, runID string, workflowCandidates map[string]struct{}, protocolCandidates map[string]struct{}) error {
	for i := range segment.Frames {
		event, ok, err := controlPlaneEventForFrame(segment.Frames[i])
		if err != nil {
			return err
		}
		if !ok || !controlPlaneEventMatchesRun(event, runID) {
			continue
		}
		collectWorkflowCandidate(event, workflowCandidates)
		collectProtocolCandidate(event, protocolCandidates)
	}
	return nil
}

func collectWorkflowCandidate(event trustpolicy.AuditEventPayload, workflowCandidates map[string]struct{}) {
	if workflow := strings.TrimSpace(workflowDefinitionHashFromAuditEvent(event)); workflow != "" {
		workflowCandidates[workflow] = struct{}{}
	}
}

func collectProtocolCandidate(event trustpolicy.AuditEventPayload, protocolCandidates map[string]struct{}) {
	if protocol, err := event.ProtocolBundleManifestHash.Identity(); err == nil && strings.TrimSpace(protocol) != "" {
		protocolCandidates[protocol] = struct{}{}
	}
}

func controlPlaneEventForFrame(frame trustpolicy.AuditSegmentRecordFrame) (trustpolicy.AuditEventPayload, bool, error) {
	envelope, err := decodeFrameEnvelope(frame)
	if err != nil {
		return trustpolicy.AuditEventPayload{}, false, err
	}
	if envelope.PayloadSchemaID != trustpolicy.AuditEventSchemaID {
		return trustpolicy.AuditEventPayload{}, false, nil
	}
	event := trustpolicy.AuditEventPayload{}
	if err := json.Unmarshal(envelope.Payload, &event); err != nil {
		return trustpolicy.AuditEventPayload{}, false, err
	}
	return event, true, nil
}

func controlPlaneEventMatchesRun(event trustpolicy.AuditEventPayload, runID string) bool {
	return strings.TrimSpace(event.Scope["run_id"]) == runID
}

func workflowDefinitionHashFromAuditEvent(event trustpolicy.AuditEventPayload) string {
	for i := range event.RelatedRefs {
		ref := event.RelatedRefs[i]
		if strings.TrimSpace(ref.ObjectFamily) != "workflow_definition" {
			continue
		}
		if strings.TrimSpace(ref.RefRole) != "binding" {
			continue
		}
		identity, err := ref.Digest.Identity()
		if err == nil {
			return strings.TrimSpace(identity)
		}
	}
	return ""
}

func (l *Ledger) evidenceBundleVerifierAndTrustPolicyDigestsLocked() (string, string, error) {
	inputs, err := l.loadVerificationContractInputsOnlyLocked()
	if err != nil {
		return "", "", err
	}
	if len(inputs.verifierRecords) == 0 {
		return "", "", nil
	}
	verifierDigest, err := canonicalDigest(inputs.verifierRecords)
	if err != nil {
		return "", "", err
	}
	verifierIdentity, err := verifierDigest.Identity()
	if err != nil {
		return "", "", err
	}
	trustPolicyDigest, err := canonicalDigest(inputs.catalog)
	if err != nil {
		return "", "", err
	}
	trustPolicyIdentity, err := trustPolicyDigest.Identity()
	if err != nil {
		return "", "", err
	}
	return verifierIdentity, trustPolicyIdentity, nil
}

func firstIdentityFromSet(values map[string]struct{}) string {
	if len(values) != 1 {
		return ""
	}
	for value := range values {
		return strings.TrimSpace(value)
	}
	return ""
}
