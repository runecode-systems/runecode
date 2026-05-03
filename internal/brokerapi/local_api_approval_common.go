package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

type approvalRecord struct {
	Summary                ApprovalSummary
	RequestEnvelope        *trustpolicy.SignedObjectEnvelope
	DecisionEnvelope       *trustpolicy.SignedObjectEnvelope
	SourceDigest           string
	ManifestHash           string
	ActionRequestHash      string
	RelevantArtifactHashes []string
}

const (
	approvalChangesIfApprovedDefault = "Promote reviewed file excerpts for downstream use."
	approvalDefaultAssuranceLevel    = "session_authenticated"
	approvalDefaultPresenceMode      = "os_confirmation"
)

func approvalIDFromRequest(envelope trustpolicy.SignedObjectEnvelope) (string, error) {
	canonical, err := jsoncanonicalizer.Transform(envelope.Payload)
	if err != nil {
		return "", fmt.Errorf("canonicalize approval request payload: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func signedEnvelopeDigest(envelope trustpolicy.SignedObjectEnvelope) (string, error) {
	b, err := json.Marshal(envelope)
	if err != nil {
		return "", err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func decodeDecisionString(payload []byte, field string, fallback string) string {
	value := map[string]any{}
	if err := json.Unmarshal(payload, &value); err != nil {
		return fallback
	}
	v, ok := value[field].(string)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func ptrArtifactSummary(value ArtifactSummary) *ArtifactSummary {
	v := value
	return &v
}

func shaDigestIdentity(input string) string {
	sum := sha256.Sum256([]byte(input))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (s *Service) approvalRecordsByID() map[string]approvalRecord {
	recs := s.ApprovalList()
	all := make(map[string]approvalRecord, len(recs))
	for _, rec := range recs {
		all[rec.ApprovalID] = approvalRecordFromStore(rec)
	}
	return all
}

func approvalRecordFromStore(record artifacts.ApprovalRecord) approvalRecord {
	summary := approvalSummaryFromStore(record)
	result := approvalRecord{
		Summary:                summary,
		RequestEnvelope:        record.RequestEnvelope,
		DecisionEnvelope:       record.DecisionEnvelope,
		SourceDigest:           record.SourceDigest,
		ManifestHash:           record.ManifestHash,
		ActionRequestHash:      record.ActionRequestHash,
		RelevantArtifactHashes: append([]string{}, record.RelevantArtifactHashes...),
	}
	populateApprovalEvidenceDigests(&result)
	return result
}

func populateApprovalEvidenceDigests(record *approvalRecord) {
	if record == nil {
		return
	}
	populateApprovalEvidenceSummaryDigests(record)
	populateApprovalConsumptionDigests(record)
}

func populateApprovalEvidenceSummaryDigests(record *approvalRecord) {
	if strings.TrimSpace(record.Summary.ScopeDigest) == "" {
		record.Summary.ScopeDigest = approvalScopeDigest(record.Summary.BoundScope)
	}
	if strings.TrimSpace(record.Summary.ArtifactSetDigest) == "" {
		record.Summary.ArtifactSetDigest = approvalArtifactSetDigest(record.RelevantArtifactHashes)
	}
	if strings.TrimSpace(record.Summary.DiffDigest) == "" {
		record.Summary.DiffDigest = approvalDiffDigest(record.SourceDigest)
	}
	if strings.TrimSpace(record.Summary.SummaryPreviewDigest) == "" {
		record.Summary.SummaryPreviewDigest = approvalSummaryPreviewDigest(record.RequestEnvelope)
	}
}

func populateApprovalConsumptionDigests(record *approvalRecord) {
	if strings.TrimSpace(record.Summary.Status) == "consumed" {
		if strings.TrimSpace(record.Summary.ConsumedActionHash) == "" && isSHA256Digest(strings.TrimSpace(record.ActionRequestHash)) {
			record.Summary.ConsumedActionHash = strings.TrimSpace(record.ActionRequestHash)
		}
		if strings.TrimSpace(record.Summary.ConsumedArtifactDigest) == "" && isSHA256Digest(strings.TrimSpace(record.SourceDigest)) {
			record.Summary.ConsumedArtifactDigest = strings.TrimSpace(record.SourceDigest)
		}
	}
	if strings.TrimSpace(record.Summary.ConsumptionLinkDigest) == "" {
		record.Summary.ConsumptionLinkDigest = approvalConsumptionLinkDigest(record.Summary)
	}
}

func approvalSummaryFromStore(record artifacts.ApprovalRecord) ApprovalSummary {
	bound := approvalBoundScopeFromStore(record)
	summary := ApprovalSummary{
		SchemaID:               "runecode.protocol.v0.ApprovalSummary",
		SchemaVersion:          "0.1.0",
		ApprovalID:             record.ApprovalID,
		Status:                 record.Status,
		RequestedAt:            record.RequestedAt.UTC().Format(time.RFC3339),
		ApprovalTriggerCode:    record.ApprovalTriggerCode,
		ChangesIfApproved:      record.ChangesIfApproved,
		ApprovalAssuranceLevel: record.ApprovalAssuranceLevel,
		PresenceMode:           record.PresenceMode,
		BoundScope:             bound,
		PolicyDecisionHash:     record.PolicyDecisionHash,
		SupersededByApprovalID: record.SupersededByApprovalID,
		RequestDigest:          record.RequestDigest,
		DecisionDigest:         record.DecisionDigest,
		ScopeDigest:            record.ScopeDigest,
		ArtifactSetDigest:      record.ArtifactSetDigest,
		DiffDigest:             record.DiffDigest,
		SummaryPreviewDigest:   record.SummaryPreviewDigest,
		ConsumedActionHash:     record.ConsumedActionHash,
		ConsumedArtifactDigest: record.ConsumedArtifactDigest,
		ConsumptionLinkDigest:  record.ConsumptionLinkDigest,
	}
	temp := approvalRecord{Summary: summary, SourceDigest: record.SourceDigest, RelevantArtifactHashes: append([]string{}, record.RelevantArtifactHashes...), RequestEnvelope: record.RequestEnvelope}
	populateApprovalEvidenceDigests(&temp)
	summary = temp.Summary
	applyOptionalApprovalSummaryTimes(&summary, record)
	return summary
}

func approvalScopeDigest(bound ApprovalBoundScope) string {
	payload := map[string]any{
		"workspace_id":         strings.TrimSpace(bound.WorkspaceID),
		"instance_id":          strings.TrimSpace(bound.InstanceID),
		"run_id":               strings.TrimSpace(bound.RunID),
		"stage_id":             strings.TrimSpace(bound.StageID),
		"step_id":              strings.TrimSpace(bound.StepID),
		"role_instance_id":     strings.TrimSpace(bound.RoleInstanceID),
		"action_kind":          strings.TrimSpace(bound.ActionKind),
		"policy_decision_hash": strings.TrimSpace(bound.PolicyDecisionHash),
	}
	return approvalCanonicalDigestIdentity(payload)
}

func approvalArtifactSetDigest(values []string) string {
	artifacts := uniqueSortedDigests(values)
	if len(artifacts) == 0 {
		return ""
	}
	return approvalCanonicalDigestIdentity(map[string]any{"relevant_artifact_hashes": artifacts})
}

func approvalDiffDigest(sourceDigest string) string {
	if isSHA256Digest(strings.TrimSpace(sourceDigest)) {
		return strings.TrimSpace(sourceDigest)
	}
	return ""
}

func approvalSummaryPreviewDigest(request *trustpolicy.SignedObjectEnvelope) string {
	if request == nil {
		return ""
	}
	payload, err := decodeApprovalRequestPayload(*request)
	if err != nil {
		return ""
	}
	details, _ := payload["details"].(map[string]any)
	if details == nil {
		return ""
	}
	if digest, err := digestIdentityFromPayloadObject(details, "stage_summary_hash"); err == nil {
		return digest
	}
	if bound, _ := details["bound_remote_mutation"].(map[string]any); bound != nil {
		if digest, err := digestIdentityFromPayloadObject(bound, "expected_result_tree_hash"); err == nil {
			return digest
		}
	}
	return ""
}

func approvalConsumptionLinkDigest(summary ApprovalSummary) string {
	if strings.TrimSpace(summary.ConsumedActionHash) == "" && strings.TrimSpace(summary.ConsumedArtifactDigest) == "" {
		return ""
	}
	payload := map[string]any{
		"approval_id":              strings.TrimSpace(summary.ApprovalID),
		"request_digest":           strings.TrimSpace(summary.RequestDigest),
		"decision_digest":          strings.TrimSpace(summary.DecisionDigest),
		"consumed_action_hash":     strings.TrimSpace(summary.ConsumedActionHash),
		"consumed_artifact_digest": strings.TrimSpace(summary.ConsumedArtifactDigest),
	}
	return approvalCanonicalDigestIdentity(payload)
}

func approvalCanonicalDigestIdentity(value any) string {
	b, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func approvalBoundScopeFromStore(record artifacts.ApprovalRecord) ApprovalBoundScope {
	return ApprovalBoundScope{
		SchemaID:           "runecode.protocol.v0.ApprovalBoundScope",
		SchemaVersion:      "0.1.0",
		WorkspaceID:        record.WorkspaceID,
		InstanceID:         record.InstanceID,
		RunID:              record.RunID,
		StageID:            record.StageID,
		StepID:             record.StepID,
		RoleInstanceID:     record.RoleInstanceID,
		ActionKind:         record.ActionKind,
		PolicyDecisionHash: record.PolicyDecisionHash,
	}
}

func applyOptionalApprovalSummaryTimes(summary *ApprovalSummary, record artifacts.ApprovalRecord) {
	if record.ExpiresAt != nil {
		summary.ExpiresAt = record.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if record.DecidedAt != nil {
		summary.DecidedAt = record.DecidedAt.UTC().Format(time.RFC3339)
	}
	if record.ConsumedAt != nil {
		summary.ConsumedAt = record.ConsumedAt.UTC().Format(time.RFC3339)
	}
}
