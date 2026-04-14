package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

const (
	approvalChangesIfApprovedDefault = "Action may proceed once approval is granted."
	approvalDefaultAssuranceLevel    = "session_authenticated"
	approvalDefaultPresenceMode      = "os_confirmation"
)

func (s *Store) ApprovalList() []ApprovalRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ApprovalRecord, 0, len(s.state.Approvals))
	for _, rec := range s.state.Approvals {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RequestedAt.Equal(out[j].RequestedAt) {
			return out[i].ApprovalID < out[j].ApprovalID
		}
		return out[i].RequestedAt.After(out[j].RequestedAt)
	})
	return out
}

func (s *Store) ApprovalGet(approvalID string) (ApprovalRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.state.Approvals[approvalID]
	return rec, ok
}

func (s *Store) RecordApproval(record ApprovalRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recordApprovalLocked(record, nil)
}

func (s *Store) RecordApprovalWithRunnerMirror(record ApprovalRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	mirror, err := runnerApprovalFromCanonicalRecord(record, s.nowFn)
	if err != nil {
		return err
	}
	return s.recordApprovalLocked(record, mirror)
}

func (s *Store) recordApprovalLocked(record ApprovalRecord, mirror *RunnerApproval) error {
	if err := s.ensureApprovalPolicyDecisionLinkLocked(&record); err != nil {
		return err
	}
	if err := requirePolicyDecisionHashForBoundApproval(record); err != nil {
		return err
	}
	if err := validateApprovalRecord(record); err != nil {
		return err
	}
	existing, exists := s.state.Approvals[record.ApprovalID]
	if exists && existing.ApprovalID == record.ApprovalID {
		// Preserve original creation-time audit linkage for idempotent updates.
		if record.AuditEventSeq == 0 {
			record.AuditEventSeq = existing.AuditEventSeq
			record.AuditEventType = existing.AuditEventType
		}
	}
	if mirror == nil {
		s.state.Approvals[record.ApprovalID] = record
		rebuildRunApprovalRefsLocked(&s.state)
		return s.saveStateLocked()
	}
	return s.recordApprovalWithRunnerMirrorLocked(record, *mirror)
}

func requirePolicyDecisionHashForBoundApproval(record ApprovalRecord) error {
	if !approvalHasBindingKeys(&record) {
		return nil
	}
	if strings.TrimSpace(record.PolicyDecisionHash) == "" {
		return ErrApprovalPolicyDecisionRequired
	}
	return nil
}

func (s *Store) ensureApprovalPolicyDecisionLinkLocked(record *ApprovalRecord) error {
	if record == nil {
		return fmt.Errorf("approval record is required")
	}
	if !approvalHasBindingKeys(record) {
		record.PolicyDecisionHash = validOrEmptyPolicyDecisionHash(record.PolicyDecisionHash, s.state.PolicyDecisions)
		return nil
	}
	matches := matchingPolicyDecisionDigests(s.state.PolicyDecisions, record.ManifestHash, record.ActionRequestHash)
	resolved, ok := resolveApprovalPolicyDecisionHash(record.PolicyDecisionHash, matches, s.state.PolicyDecisions)
	if ok {
		record.PolicyDecisionHash = resolved
		return nil
	}
	record.PolicyDecisionHash = ""
	return nil
}

func rebuildRunApprovalRefsLocked(state *StoreState) {
	state.RunApprovalRefs = map[string][]string{}
	for _, rec := range state.Approvals {
		if rec.RunID == "" {
			continue
		}
		state.RunApprovalRefs[rec.RunID] = uniqueSortedStrings(append(state.RunApprovalRefs[rec.RunID], rec.ApprovalID))
	}
}

func (s *Store) reconcileApprovalPolicyDecisionLinksLocked() (bool, error) {
	changed := false
	for approvalID, rec := range s.state.Approvals {
		before := rec.PolicyDecisionHash
		if err := s.ensureApprovalPolicyDecisionLinkLocked(&rec); err != nil {
			return false, fmt.Errorf("approval %q policy decision linkage: %w", approvalID, err)
		}
		if err := requirePolicyDecisionHashForBoundApproval(rec); err != nil {
			return false, fmt.Errorf("approval %q policy decision linkage: %w", approvalID, err)
		}
		if rec.PolicyDecisionHash != before {
			s.state.Approvals[approvalID] = rec
			changed = true
		}
	}
	return changed, nil
}

func buildApprovalFromPolicyDecision(record PolicyDecisionRecord, nowFn func() time.Time) (ApprovalRecord, bool, error) {
	if record.DecisionOutcome != "require_human_approval" {
		return ApprovalRecord{}, false, nil
	}
	required, err := requiredApprovalPayload(record)
	if err != nil {
		return ApprovalRecord{}, false, err
	}
	requestedAt, expiresAt := approvalTiming(record, required, nowFn)
	summary := approvalDecisionSummary(record, required)
	requestEnvelope, requestDigest, err := approvalDecisionRequestEnvelope(record, requestedAt, expiresAt, summary)
	if err != nil {
		return ApprovalRecord{}, false, err
	}
	approval := approvalRecordFromDecisionSummary(record, summary, requestEnvelope, requestDigest, requestedAt, expiresAt)
	if err := validateApprovalRecord(approval); err != nil {
		return ApprovalRecord{}, false, err
	}
	return approval, true, nil
}

func requiredApprovalPayload(record PolicyDecisionRecord) (map[string]any, error) {
	if record.RequiredApproval == nil {
		return nil, fmt.Errorf("required_approval payload missing")
	}
	return record.RequiredApproval, nil
}

func approvalDecisionRequestEnvelope(record PolicyDecisionRecord, requestedAt, expiresAt time.Time, summary approvalDecisionDerivedSummary) (trustpolicy.SignedObjectEnvelope, string, error) {
	requestPayload, err := approvalRequestPayloadFromDecision(record, requestedAt, expiresAt, summary.trigger, summary.changes, summary.assurance, summary.presence, summary.runID, summary.stepID)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, "", err
	}
	return approvalRequestEnvelopeAndDigest(requestPayload)
}

func approvalRecordFromDecisionSummary(record PolicyDecisionRecord, summary approvalDecisionDerivedSummary, requestEnvelope trustpolicy.SignedObjectEnvelope, requestDigest string, requestedAt, expiresAt time.Time) ApprovalRecord {
	expiresCopy := expiresAt.UTC()
	return ApprovalRecord{
		ApprovalID:             requestDigest,
		Status:                 "pending",
		WorkspaceID:            summary.workspaceID,
		InstanceID:             summary.instanceID,
		RunID:                  summary.runID,
		StageID:                summary.stageID,
		StepID:                 summary.stepID,
		RoleInstanceID:         summary.roleInstanceID,
		ActionKind:             summary.actionKind,
		RequestedAt:            requestedAt,
		ExpiresAt:              &expiresCopy,
		ApprovalTriggerCode:    summary.trigger,
		ChangesIfApproved:      summary.changes,
		ApprovalAssuranceLevel: summary.assurance,
		PresenceMode:           summary.presence,
		PolicyDecisionHash:     record.Digest,
		ManifestHash:           record.ManifestHash,
		ActionRequestHash:      record.ActionRequestHash,
		RelevantArtifactHashes: summary.relevant,
		RequestDigest:          requestDigest,
		SourceDigest:           summary.sourceDigest,
		RequestEnvelope:        &requestEnvelope,
	}
}

func approvalRequestEnvelopeAndDigest(payload map[string]any) (trustpolicy.SignedObjectEnvelope, string, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, "", fmt.Errorf("marshal approval request payload: %w", err)
	}
	canonical, err := jsoncanonicalizer.Transform(payloadBytes)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, "", fmt.Errorf("canonicalize approval request payload: %w", err)
	}
	sum := sha256.Sum256(canonical)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	envelope := trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.ApprovalRequestSchemaID,
		PayloadSchemaVersion: trustpolicy.ApprovalRequestSchemaVersion,
		Payload:              payloadBytes,
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature: trustpolicy.SignatureBlock{
			Alg:        "ed25519",
			KeyID:      trustpolicy.KeyIDProfile,
			KeyIDValue: strings.Repeat("0", 64),
			Signature:  "cGVuZGluZw==",
		},
	}
	return envelope, digest, nil
}
