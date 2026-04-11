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
		RunID:              record.RunID,
		StageID:            record.StageID,
		StepID:             record.StepID,
		RoleInstanceID:     record.RoleInstanceID,
		ActionKind:         record.ActionKind,
		PolicyDecisionHash: record.PolicyDecisionHash,
	}
}

func (s *Service) approvalDetailFromRecord(record approvalRecord) (ApprovalDetail, error) {
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	detail := ApprovalDetail{
		SchemaID:              "runecode.protocol.v0.ApprovalDetail",
		SchemaVersion:         "0.1.0",
		ApprovalID:            record.Summary.ApprovalID,
		LifecycleDetail:       approvalLifecycleDetailFromRecord(record, now),
		WhatChangesIfApproved: approvalWhatChangesIfApprovedFromRecord(record),
		BlockedWorkScope:      approvalBlockedWorkScopeFromRecord(record),
	}
	if policyRef := strings.TrimSpace(record.Summary.PolicyDecisionHash); policyRef != "" {
		if decision, ok := s.PolicyDecisionGet(policyRef); ok {
			detail.PolicyReasonCode = strings.TrimSpace(decision.PolicyReasonCode)
		}
	}
	if strings.TrimSpace(record.Summary.BoundScope.ActionKind) == "stage_summary_sign_off" {
		detail.BindingKind = "stage_sign_off"
		detail.BoundStageSummaryHash = stageSummaryHashForApprovalDetail(record)
		if detail.BoundStageSummaryHash == "" {
			return ApprovalDetail{}, fmt.Errorf("stage sign-off approval missing bound stage_summary_hash")
		}
		detail.BoundIdentity = approvalBoundIdentityFromRecord(record, detail.BindingKind, "", detail.BoundStageSummaryHash)
		return detail, nil
	}
	detail.BindingKind = "exact_action"
	detail.BoundActionHash = record.ActionRequestHash
	if strings.TrimSpace(detail.BoundActionHash) == "" {
		return ApprovalDetail{}, fmt.Errorf("exact-action approval missing bound action hash")
	}
	detail.BoundIdentity = approvalBoundIdentityFromRecord(record, detail.BindingKind, detail.BoundActionHash, "")
	return detail, nil
}

func approvalLifecycleDetailFromRecord(record approvalRecord, now time.Time) ApprovalLifecycleDetail {
	state := strings.TrimSpace(record.Summary.Status)
	detail := ApprovalLifecycleDetail{
		SchemaID:            "runecode.protocol.v0.ApprovalLifecycleDetail",
		SchemaVersion:       "0.1.0",
		LifecycleState:      state,
		LifecycleReasonCode: lifecycleReasonCodeForApprovalSummary(record.Summary),
		Stale:               approvalIsStale(record, now),
		StaleReasonCode:     staleReasonCodeForApprovalRecord(record, now),
		ExpiresAt:           strings.TrimSpace(record.Summary.ExpiresAt),
		DecidedAt:           strings.TrimSpace(record.Summary.DecidedAt),
		ConsumedAt:          strings.TrimSpace(record.Summary.ConsumedAt),
	}
	if state == "superseded" {
		detail.SupersededByApprovalID = strings.TrimSpace(record.Summary.SupersededByApprovalID)
	}
	if !detail.Stale {
		detail.StaleReasonCode = ""
	}
	return detail
}

func lifecycleReasonCodeForApprovalSummary(summary ApprovalSummary) string {
	switch strings.TrimSpace(summary.Status) {
	case "pending":
		return "approval_pending"
	case "approved":
		return "approval_approved"
	case "denied":
		return "approval_denied"
	case "expired":
		return "approval_expired"
	case "cancelled":
		return "approval_cancelled"
	case "superseded":
		return "approval_superseded"
	case "consumed":
		return "approval_consumed"
	default:
		return "approval_lifecycle_unknown"
	}
}

func approvalIsStale(record approvalRecord, now time.Time) bool {
	state := strings.TrimSpace(record.Summary.Status)
	return state == "superseded" || state == "expired" || staleReasonCodeForApprovalRecord(record, now) != ""
}

func staleReasonCodeForApprovalRecord(record approvalRecord, now time.Time) string {
	if strings.TrimSpace(record.Summary.Status) == "superseded" {
		return "approval_superseded"
	}
	if strings.TrimSpace(record.Summary.Status) == "expired" {
		return "approval_expired"
	}
	if isApprovalExpiredAt(record, now) {
		return "approval_expired"
	}
	return ""
}

func isApprovalExpiredAt(record approvalRecord, at time.Time) bool {
	expiresAt, ok := parseRFC3339(strings.TrimSpace(record.Summary.ExpiresAt))
	if !ok {
		return false
	}
	return !at.Before(expiresAt)
}

func approvalWhatChangesIfApprovedFromRecord(record approvalRecord) ApprovalWhatChangesIfApproved {
	summary := strings.TrimSpace(record.Summary.ChangesIfApproved)
	if summary == "" {
		summary = approvalChangesIfApprovedDefault
	}
	return ApprovalWhatChangesIfApproved{
		SchemaID:      "runecode.protocol.v0.ApprovalWhatChangesIfApproved",
		SchemaVersion: "0.1.0",
		Summary:       summary,
		EffectKind:    approvalEffectKindForActionKind(record.Summary.BoundScope.ActionKind),
	}
}

func approvalBlockedWorkScopeFromRecord(record approvalRecord) ApprovalBlockedWorkScope {
	bound := record.Summary.BoundScope
	return ApprovalBlockedWorkScope{
		SchemaID:       "runecode.protocol.v0.ApprovalBlockedWorkScope",
		SchemaVersion:  "0.1.0",
		ScopeKind:      approvalBlockedScopeKind(bound),
		WorkspaceID:    bound.WorkspaceID,
		RunID:          bound.RunID,
		StageID:        bound.StageID,
		StepID:         bound.StepID,
		RoleInstanceID: bound.RoleInstanceID,
		ActionKind:     bound.ActionKind,
	}
}

func approvalBoundIdentityFromRecord(record approvalRecord, bindingKind, boundActionHash, boundStageSummaryHash string) ApprovalBoundIdentity {
	identity := ApprovalBoundIdentity{
		SchemaID:               "runecode.protocol.v0.ApprovalBoundIdentity",
		SchemaVersion:          "0.1.0",
		ApprovalRequestDigest:  strings.TrimSpace(record.Summary.RequestDigest),
		ApprovalDecisionDigest: strings.TrimSpace(record.Summary.DecisionDigest),
		PolicyDecisionHash:     strings.TrimSpace(record.Summary.PolicyDecisionHash),
		ManifestHash:           strings.TrimSpace(record.ManifestHash),
		RelevantArtifactHashes: append([]string{}, record.RelevantArtifactHashes...),
		BindingKind:            bindingKind,
		BoundActionHash:        strings.TrimSpace(boundActionHash),
		BoundStageSummaryHash:  strings.TrimSpace(boundStageSummaryHash),
	}
	if identity.ApprovalRequestDigest == "" {
		identity.ApprovalRequestDigest = strings.TrimSpace(record.Summary.ApprovalID)
	}
	if decision := approvalDecisionFromEnvelope(record.DecisionEnvelope); decision != nil {
		identity.DecisionApprover = &decision.Approver
	}
	if record.DecisionEnvelope != nil {
		identity.DecisionVerifierKeyID = strings.TrimSpace(record.DecisionEnvelope.Signature.KeyID)
		identity.DecisionVerifierKeyIDValue = strings.TrimSpace(record.DecisionEnvelope.Signature.KeyIDValue)
	}
	return identity
}

func approvalEffectKindForActionKind(actionKind string) string {
	switch strings.TrimSpace(actionKind) {
	case "stage_summary_sign_off":
		return "stage_sign_off"
	case "promotion":
		return "promotion"
	default:
		return "action_execution"
	}
}

func approvalBlockedScopeKind(bound ApprovalBoundScope) string {
	if strings.TrimSpace(bound.StepID) != "" {
		return "step"
	}
	if strings.TrimSpace(bound.StageID) != "" {
		return "stage"
	}
	if strings.TrimSpace(bound.RunID) != "" {
		return "run"
	}
	if strings.TrimSpace(bound.WorkspaceID) != "" {
		return "workspace"
	}
	return "action_kind"
}

func approvalDecisionFromEnvelope(envelope *trustpolicy.SignedObjectEnvelope) *trustpolicy.ApprovalDecision {
	if envelope == nil {
		return nil
	}
	decoded, err := decodeApprovalDecision(*envelope)
	if err != nil {
		return nil
	}
	return &decoded
}

func stageSummaryHashForApprovalDetail(record approvalRecord) string {
	if record.RequestEnvelope == nil {
		return ""
	}
	payload := map[string]any{}
	if err := json.Unmarshal(record.RequestEnvelope.Payload, &payload); err != nil {
		return ""
	}
	details, _ := payload["details"].(map[string]any)
	if details == nil {
		return ""
	}
	digest, err := digestIdentityFromPayloadObject(details, "stage_summary_hash")
	if err != nil {
		return ""
	}
	return digest
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
