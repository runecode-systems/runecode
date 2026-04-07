package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

type approvalRecord struct {
	Summary          ApprovalSummary
	RequestEnvelope  *trustpolicy.SignedObjectEnvelope
	DecisionEnvelope *trustpolicy.SignedObjectEnvelope
}

type approvalState struct {
	mu      sync.Mutex
	records map[string]approvalRecord
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
	all := s.derivedApprovalRecords()

	s.approvals.mu.Lock()
	defer s.approvals.mu.Unlock()
	pruneDerivedApprovalOverlays(all, approvalOverlayDigests(s.approvals.records))
	for id, record := range s.approvals.records {
		all[id] = record
	}
	return all
}

func (s *Service) derivedApprovalRecords() map[string]approvalRecord {
	now := time.Now().UTC()
	byID := map[string]approvalRecord{}
	unapprovedByDigest, consumedUnapproved, approvedRecords := classifyApprovalArtifacts(s.List())
	addResolvedApprovalRecords(byID, approvedRecords, unapprovedByDigest)
	addPendingApprovalRecords(byID, unapprovedByDigest, consumedUnapproved, now)
	return byID
}

func inferredPendingApprovalRecord(record artifacts.ArtifactRecord, now time.Time) approvalRecord {
	approvalID := shaDigestIdentity("pending-approval:" + record.Reference.Digest)
	requestedAt := nonZeroRecordTime(record.CreatedAt, now)
	expiresAt := requestedAt.Add(30 * time.Minute)
	workspaceID := workspaceIDForRun(record.RunID)
	if record.RunID == "" {
		workspaceID = "workspace-local"
	}
	return approvalRecord{Summary: ApprovalSummary{
		SchemaID:               "runecode.protocol.v0.ApprovalSummary",
		SchemaVersion:          "0.1.0",
		ApprovalID:             approvalID,
		Status:                 "pending",
		RequestedAt:            requestedAt.Format(time.RFC3339),
		ExpiresAt:              expiresAt.Format(time.RFC3339),
		ApprovalTriggerCode:    "excerpt_promotion",
		ChangesIfApproved:      approvalChangesIfApprovedDefault,
		ApprovalAssuranceLevel: approvalDefaultAssuranceLevel,
		PresenceMode:           approvalDefaultPresenceMode,
		BoundScope: ApprovalBoundScope{
			SchemaID:      "runecode.protocol.v0.ApprovalBoundScope",
			SchemaVersion: "0.1.0",
			WorkspaceID:   workspaceID,
			RunID:         record.RunID,
			StageID:       stageIDForRun(record.RunID),
			StepID:        record.StepID,
			ActionKind:    "excerpt_promotion",
		},
		RequestDigest: approvalID,
	}}
}

func inferredResolvedApprovalRecord(record artifacts.ArtifactRecord, source artifacts.ArtifactRecord, hasSource bool) approvalRecord {
	if strings.TrimSpace(record.PromotionRequestHash) == "" {
		return approvalRecord{}
	}
	requestedAt, decidedAt := resolvedApprovalTimes(record, source, hasSource)
	boundScope := resolvedApprovalScope(record)
	return approvalRecord{Summary: ApprovalSummary{
		SchemaID:               "runecode.protocol.v0.ApprovalSummary",
		SchemaVersion:          "0.1.0",
		ApprovalID:             record.PromotionRequestHash,
		Status:                 "approved",
		RequestedAt:            requestedAt,
		DecidedAt:              decidedAt,
		ApprovalTriggerCode:    "excerpt_promotion",
		ChangesIfApproved:      approvalChangesIfApprovedDefault,
		ApprovalAssuranceLevel: approvalDefaultAssuranceLevel,
		PresenceMode:           approvalDefaultPresenceMode,
		BoundScope:             boundScope,
		PolicyDecisionHash:     record.ApprovalDecisionHash,
		RequestDigest:          record.PromotionRequestHash,
		DecisionDigest:         record.ApprovalDecisionHash,
	}}
}

func nonZeroRecordTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value.UTC()
}
