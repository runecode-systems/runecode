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
	return approvalRecord{
		Summary:                summary,
		RequestEnvelope:        record.RequestEnvelope,
		DecisionEnvelope:       record.DecisionEnvelope,
		SourceDigest:           record.SourceDigest,
		ManifestHash:           record.ManifestHash,
		ActionRequestHash:      record.ActionRequestHash,
		RelevantArtifactHashes: append([]string{}, record.RelevantArtifactHashes...),
	}
}

func approvalSummaryFromStore(record artifacts.ApprovalRecord) ApprovalSummary {
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
		BoundScope:             approvalBoundScopeFromStore(record),
		PolicyDecisionHash:     record.PolicyDecisionHash,
		SupersededByApprovalID: record.SupersededByApprovalID,
		RequestDigest:          record.RequestDigest,
		DecisionDigest:         record.DecisionDigest,
	}
	applyOptionalApprovalSummaryTimes(&summary, record)
	return summary
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
