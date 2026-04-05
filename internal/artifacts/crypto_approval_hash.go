package artifacts

import "encoding/json"

func promotionActionRequestHash(req PromotionRequest) (string, error) {
	payload, err := json.Marshal(struct {
		Approver             string `json:"approver"`
		Commit               string `json:"commit"`
		ExtractorToolVersion string `json:"extractor_tool_version"`
		RepoPath             string `json:"repo_path"`
		UnapprovedDigest     string `json:"unapproved_digest"`
	}{
		Approver:             req.Approver,
		Commit:               req.Commit,
		ExtractorToolVersion: req.ExtractorToolVersion,
		RepoPath:             req.RepoPath,
		UnapprovedDigest:     req.UnapprovedDigest,
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
