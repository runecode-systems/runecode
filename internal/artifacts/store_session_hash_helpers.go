package artifacts

import (
	"encoding/json"
	"strings"
)

func SessionSendMessageIdempotencyHash(sessionID, role, contentText string, links SessionTranscriptLinksDurableState) (string, error) {
	payload := map[string]any{
		"session_id":   strings.TrimSpace(sessionID),
		"role":         strings.TrimSpace(role),
		"content_text": strings.TrimSpace(contentText),
		"related_links": map[string]any{
			"run_ids":              append([]string{}, links.RunIDs...),
			"approval_ids":         append([]string{}, links.ApprovalIDs...),
			"artifact_digests":     append([]string{}, links.ArtifactDigests...),
			"audit_record_digests": append([]string{}, links.AuditRecordDigests...),
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalizeJSONBytes(b)
	if err != nil {
		return "", err
	}
	return digestBytes(canonical), nil
}

func SessionExecutionTriggerIdempotencyHash(sessionID, triggerSource, requestedOperation, approvalProfile, autonomyPosture, userMessageContentText string, workflowRouting SessionWorkflowPackRoutingDurableState) (string, error) {
	boundInputs := make([]map[string]any, 0, len(workflowRouting.BoundInputArtifacts))
	for _, artifact := range workflowRouting.BoundInputArtifacts {
		boundInputs = append(boundInputs, map[string]any{
			"artifact_ref":    strings.TrimSpace(artifact.ArtifactRef),
			"artifact_digest": strings.TrimSpace(artifact.ArtifactDigest),
		})
	}
	payload := map[string]any{
		"session_id":          strings.TrimSpace(sessionID),
		"trigger_source":      strings.TrimSpace(triggerSource),
		"requested_operation": strings.TrimSpace(requestedOperation),
		"approval_profile":    strings.TrimSpace(approvalProfile),
		"autonomy_posture":    strings.TrimSpace(autonomyPosture),
		"workflow_routing": map[string]any{
			"workflow_family":       strings.TrimSpace(workflowRouting.WorkflowFamily),
			"workflow_operation":    strings.TrimSpace(workflowRouting.WorkflowOperation),
			"bound_input_artifacts": boundInputs,
		},
		"user_message_content_text": strings.TrimSpace(userMessageContentText),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalizeJSONBytes(b)
	if err != nil {
		return "", err
	}
	return digestBytes(canonical), nil
}
