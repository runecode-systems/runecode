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
