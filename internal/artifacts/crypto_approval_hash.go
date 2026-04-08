package artifacts

import "encoding/json"

const (
	actionRequestSchemaID        = "runecode.protocol.v0.ActionRequest"
	actionRequestSchemaVersion   = "0.1.0"
	actionKindPromotion          = "promotion"
	actionPayloadPromotionSchema = "runecode.protocol.v0.ActionPayloadPromotion"
)

func promotionActionRequestHash(req PromotionRequest) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"schema_id":                actionRequestSchemaID,
		"schema_version":           actionRequestSchemaVersion,
		"action_kind":              actionKindPromotion,
		"capability_id":            actionKindPromotion,
		"action_payload_schema_id": actionPayloadPromotionSchema,
		"action_payload": map[string]any{
			"schema_id":              actionPayloadPromotionSchema,
			"schema_version":         "0.1.0",
			"promotion_kind":         "excerpt",
			"source_artifact_hash":   req.UnapprovedDigest,
			"target_data_class":      "approved_file_excerpts",
			"justification":          "promotion approval request",
			"repo_path":              req.RepoPath,
			"commit":                 req.Commit,
			"extractor_tool_version": req.ExtractorToolVersion,
			"approver":               req.Approver,
		},
	})
	if err != nil {
		return "", err
	}
	canonical, err := canonicalizeJSONBytes(payload)
	if err != nil {
		return "", err
	}
	return digestBytes(canonical), nil
}
