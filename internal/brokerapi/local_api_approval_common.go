package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
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

func uniqueSortedDigests(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if !isSHA256Digest(trimmed) {
			continue
		}
		set[trimmed] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func isSHA256Digest(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, c := range value[len("sha256:"):] {
		if (c < 'a' || c > 'f') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func (s *Service) approvalRecordsByID() map[string]approvalRecord {
	all := s.derivedApprovalRecords()

	s.approvals.mu.Lock()
	defer s.approvals.mu.Unlock()
	overlayDigests := map[string]struct{}{}
	for _, record := range s.approvals.records {
		if isSHA256Digest(record.Summary.RequestDigest) {
			overlayDigests[record.Summary.RequestDigest] = struct{}{}
		}
		if isSHA256Digest(record.Summary.DecisionDigest) {
			overlayDigests[record.Summary.DecisionDigest] = struct{}{}
		}
	}
	for id, record := range all {
		if _, ok := overlayDigests[record.Summary.RequestDigest]; ok {
			delete(all, id)
			continue
		}
		if _, ok := overlayDigests[record.Summary.DecisionDigest]; ok {
			delete(all, id)
		}
	}
	for id, record := range s.approvals.records {
		all[id] = record
	}
	return all
}

func (s *Service) derivedApprovalRecords() map[string]approvalRecord {
	records := s.List()
	now := time.Now().UTC()
	byID := map[string]approvalRecord{}

	unapprovedByDigest := map[string]artifacts.ArtifactRecord{}
	consumedUnapproved := map[string]struct{}{}
	approvedRecords := make([]artifacts.ArtifactRecord, 0)
	for _, record := range records {
		switch record.Reference.DataClass {
		case artifacts.DataClassUnapprovedFileExcerpts:
			if record.Reference.Digest != "" {
				unapprovedByDigest[record.Reference.Digest] = record
			}
		case artifacts.DataClassApprovedFileExcerpts:
			approvedRecords = append(approvedRecords, record)
			if record.ApprovalOfDigest != "" {
				consumedUnapproved[record.ApprovalOfDigest] = struct{}{}
			}
		}
	}

	for _, approved := range approvedRecords {
		source, hasSource := unapprovedByDigest[approved.ApprovalOfDigest]
		resolved := inferredResolvedApprovalRecord(approved, source, hasSource)
		if resolved.Summary.ApprovalID != "" {
			byID[resolved.Summary.ApprovalID] = resolved
		}
	}

	for digest, record := range unapprovedByDigest {
		if _, ok := consumedUnapproved[digest]; ok {
			continue
		}
		pending := inferredPendingApprovalRecord(record, now)
		byID[pending.Summary.ApprovalID] = pending
	}
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
	requestTime := record.CreatedAt
	if hasSource && !source.CreatedAt.IsZero() {
		requestTime = source.CreatedAt
	}
	if requestTime.IsZero() {
		requestTime = time.Now().UTC()
	}
	decidedTime := record.CreatedAt
	if record.PromotionApprovedAt != nil {
		decidedTime = record.PromotionApprovedAt.UTC()
	}
	if decidedTime.IsZero() {
		decidedTime = requestTime
	}
	requestedAt := requestTime.UTC().Format(time.RFC3339)
	decidedAt := decidedTime.UTC().Format(time.RFC3339)
	status := "approved"
	boundScope := ApprovalBoundScope{
		SchemaID:      "runecode.protocol.v0.ApprovalBoundScope",
		SchemaVersion: "0.1.0",
		WorkspaceID:   workspaceIDForRun(record.RunID),
		RunID:         record.RunID,
		StageID:       stageIDForRun(record.RunID),
		StepID:        record.StepID,
		ActionKind:    "excerpt_promotion",
	}
	return approvalRecord{Summary: ApprovalSummary{
		SchemaID:               "runecode.protocol.v0.ApprovalSummary",
		SchemaVersion:          "0.1.0",
		ApprovalID:             record.PromotionRequestHash,
		Status:                 status,
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
