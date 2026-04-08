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
	SourceDigest     string
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
	pruneDerivedApprovalOverlays(all, approvalOverlayDigests(s.approvals.records), approvalOverlaySourceDigests(s.approvals.records))
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
	requestedAt := nonZeroRecordTime(record.CreatedAt, now)
	expiresAt := requestedAt.Add(30 * time.Minute)
	requestEnvelope := inferredPendingApprovalRequestEnvelope(record, requestedAt, expiresAt)
	approvalID, err := approvalIDFromRequest(requestEnvelope)
	if err != nil {
		approvalID = shaDigestIdentity("approval-request:" + record.Reference.Digest)
	}
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
			ActionKind:    "promotion",
		},
		RequestDigest: approvalID,
	}, RequestEnvelope: &requestEnvelope, SourceDigest: record.Reference.Digest}
}

func inferredPendingApprovalRequestEnvelope(record artifacts.ArtifactRecord, requestedAt, expiresAt time.Time) trustpolicy.SignedObjectEnvelope {
	payload := inferredPendingApprovalRequestPayload(record, requestedAt, expiresAt)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		payloadBytes = []byte(`{"schema_id":"` + trustpolicy.ApprovalRequestSchemaID + `","schema_version":"` + trustpolicy.ApprovalRequestSchemaVersion + `"}`)
	}
	return trustpolicy.SignedObjectEnvelope{
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
}

func inferredPendingApprovalRequestPayload(record artifacts.ArtifactRecord, requestedAt, expiresAt time.Time) map[string]any {
	manifestHash := inferredPendingManifestHash(record)
	actionHash := strings.TrimPrefix(shaDigestIdentity("pending-action:"+record.Reference.Digest), "sha256:")
	return map[string]any{
		"schema_id":             trustpolicy.ApprovalRequestSchemaID,
		"schema_version":        trustpolicy.ApprovalRequestSchemaVersion,
		"approval_profile":      "moderate",
		"approval_trigger_code": "excerpt_promotion",
		"requester":             inferredPendingApprovalRequester(),
		"manifest_hash":         map[string]any{"hash_alg": "sha256", "hash": manifestHash},
		"action_request_hash":   map[string]any{"hash_alg": "sha256", "hash": actionHash},
		"relevant_artifact_hashes": []map[string]any{{
			"hash_alg": "sha256",
			"hash":     strings.TrimPrefix(record.Reference.Digest, "sha256:"),
		}},
		"details_schema_id": "runecode.protocol.details.approval.excerpt-promotion.v0",
		"details": map[string]any{
			"run_id":  record.RunID,
			"step_id": record.StepID,
		},
		"approval_assurance_level": "session_authenticated",
		"presence_mode":            "os_confirmation",
		"requested_at":             requestedAt.UTC().Format(time.RFC3339),
		"expires_at":               expiresAt.UTC().Format(time.RFC3339),
		"staleness_posture":        "invalidate_on_bound_input_change",
		"changes_if_approved":      approvalChangesIfApprovedDefault,
		"signatures":               inferredPendingApprovalSignatures(),
	}
}

func inferredPendingManifestHash(record artifacts.ArtifactRecord) string {
	manifestHash := strings.TrimPrefix(record.Reference.ProvenanceReceiptHash, "sha256:")
	if len(manifestHash) == 64 {
		return manifestHash
	}
	return strings.TrimPrefix(shaDigestIdentity("manifest:"+record.RunID), "sha256:")
}

func inferredPendingApprovalRequester() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.PrincipalIdentity",
		"schema_version": "0.2.0",
		"actor_kind":     "daemon",
		"principal_id":   "broker",
		"instance_id":    "broker-local",
	}
}

func inferredPendingApprovalSignatures() []map[string]any {
	return []map[string]any{{
		"alg":          "ed25519",
		"key_id":       trustpolicy.KeyIDProfile,
		"key_id_value": strings.Repeat("0", 64),
		"signature":    "cGVuZGluZw==",
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
	}, SourceDigest: record.ApprovalOfDigest}
}

func nonZeroRecordTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value.UTC()
}
