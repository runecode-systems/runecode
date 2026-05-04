package brokerapi

import (
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (s *Service) seedDevManualSession(approvalID string, auditRecordDigest string, artifactDigests []string, profile string) error {
	occurredAt, err := time.Parse(time.RFC3339, "2026-03-13T12:16:00Z")
	if err != nil {
		return err
	}
	appendSpec, err := devManualSessionAppendSpec(approvalID, auditRecordDigest, artifactDigests)
	if err != nil {
		return err
	}
	_, err = s.AppendSessionMessage(devManualSessionAppendRequest(occurredAt, appendSpec, profile))
	return err
}

type devManualSessionSpec struct {
	links           artifacts.SessionTranscriptLinksDurableState
	idempotencyHash string
}

func devManualSessionAppendSpec(approvalID string, auditRecordDigest string, artifactDigests []string) (devManualSessionSpec, error) {
	links := artifacts.SessionTranscriptLinksDurableState{
		RunIDs:             []string{devManualSeedRunID},
		ApprovalIDs:        []string{approvalID},
		ArtifactDigests:    append([]string{}, artifactDigests...),
		AuditRecordDigests: []string{auditRecordDigest},
	}
	idempotencyHash, err := artifacts.SessionSendMessageIdempotencyHash(devManualSeedSessionID, "user", "Please review and approve this run.", links)
	if err != nil {
		return devManualSessionSpec{}, err
	}
	return devManualSessionSpec{links: links, idempotencyHash: idempotencyHash}, nil
}

func devManualSessionAppendRequest(occurredAt time.Time, spec devManualSessionSpec, profile string) artifacts.SessionMessageAppendRequest {
	return artifacts.SessionMessageAppendRequest{
		SessionID:       devManualSeedSessionID,
		WorkspaceID:     devManualSeedWorkspaceID,
		CreatedByRunID:  devManualSeedRunID,
		Role:            "user",
		ContentText:     "Please review and approve this run.",
		RelatedLinks:    spec.links,
		IdempotencyKey:  "dev-seed-msg-1:" + profile,
		IdempotencyHash: spec.idempotencyHash,
		OccurredAt:      occurredAt,
	}
}

func (s *Service) ensureDevManualSessionAuditLink(recordDigest string, profile string) error {
	exists, err := s.devManualSessionAuditLinkExists(recordDigest, profile)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.AppendTrustedAuditEvent("manual_seed_link", "brokerapi", map[string]any{
		"run_id":        devManualSeedRunID,
		"session_id":    devManualSeedSessionID,
		"record_digest": recordDigest,
		"seed_profile":  profile,
	})
}

func (s *Service) devManualSessionAuditLinkExists(recordDigest string, profile string) (bool, error) {
	events, err := s.ReadAuditEvents()
	if err != nil {
		return false, err
	}
	for _, event := range events {
		if devManualSessionAuditLinkMatches(event, recordDigest, profile) {
			return true, nil
		}
	}
	return false, nil
}

func devManualSessionAuditLinkMatches(event artifacts.AuditEvent, recordDigest string, expectedProfile string) bool {
	if event.Type != "manual_seed_link" {
		return false
	}
	details := event.Details
	if details == nil {
		return false
	}
	runID, ok := details["run_id"].(string)
	if !ok || runID != devManualSeedRunID {
		return false
	}
	sessionID, ok := details["session_id"].(string)
	if !ok || sessionID != devManualSeedSessionID {
		return false
	}
	profile, ok := details["seed_profile"].(string)
	if !ok || profile != expectedProfile {
		return false
	}
	linkedDigest, ok := details["record_digest"].(string)
	return ok && linkedDigest == recordDigest
}

func devManualApprovalDecision(profile string) (policyengine.PolicyDecision, error) {
	precedence := "manual_seed_profile:" + profile
	if profile == devManualSeedDegradedProfile {
		precedence = precedence + ":degraded"
	}
	return policyengine.PolicyDecision{
		SchemaID:                 "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:            "0.3.0",
		DecisionOutcome:          "require_human_approval",
		PolicyReasonCode:         "approval_required",
		ManifestHash:             digestWithByte("e"),
		ActionRequestHash:        digestWithByte("f"),
		PolicyInputHashes:        []string{digestWithByte("7")},
		RelevantArtifactHashes:   []string{digestWithByte("8")},
		DetailsSchemaID:          "runecode.protocol.details.policy.evaluation.v0",
		Details:                  map[string]any{"precedence": precedence},
		RequiredApprovalSchemaID: "runecode.protocol.details.policy.required_approval.moderate.workspace_write.v0",
		RequiredApproval: map[string]any{
			"approval_trigger_code":    "excerpt_promotion",
			"approval_assurance_level": "moderate",
			"presence_mode":            "os_confirmation",
			"approval_ttl_seconds":     1800,
			"changes_if_approved":      "Promotion continues",
			"scope": map[string]any{
				"workspace_id":     devManualSeedWorkspaceID,
				"run_id":           devManualSeedRunID,
				"stage_id":         devManualSeedStageID,
				"role_instance_id": devManualSeedRoleInstanceID,
				"action_kind":      "promotion",
			},
		},
	}, nil
}
