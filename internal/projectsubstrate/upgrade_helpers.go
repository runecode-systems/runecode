package projectsubstrate

import "strings"

func upgradePreconditions(layout repositoryLayout) (runecontextConfig, []UpgradePrecondition, []string, bool) {
	preconditions := []UpgradePrecondition{}
	reasons := []string{}
	addPrecondition := func(code string, satisfied bool) {
		preconditions = append(preconditions, UpgradePrecondition{Code: code, Satisfied: satisfied})
		if !satisfied {
			reasons = append(reasons, code)
		}
	}
	addPrecondition(reasonRemediationConfigAnchorRequired, layout.hasConfigAnchor)
	addPrecondition(reasonRemediationInitRequired, layout.hasSourceAnchor && layout.hasAssuranceAnchor)
	addPrecondition(reasonPrivateMirrorDetected, !layout.hasPrivateTruthCopy)
	cfg, cfgErr := parseRunecontextConfig(layout.runecontextYAML)
	configValid := cfgErr == nil
	addPrecondition(reasonRemediationConfigInvalid, configValid)
	if configValid {
		addPrecondition(reasonRemediationDeclaredSourceConflicts, strings.TrimSpace(cfg.Source.Path) == CanonicalSourcePath)
	}
	return cfg, preconditions, reasons, configValid
}

func blockedUpgradePreview(preview UpgradePreview, snapshot ValidationSnapshot, reasons []string) UpgradePreview {
	preview.Status = upgradePreviewStatusBlockedRemediation
	preview.ReasonCodes = normalizeReasonCodes(append(reasons, reasonRemediationFlowRequired))
	preview.ExpectedSnapshot = snapshot
	preview.RequiredFollowUp = remediationFollowUpForReasons(preview.ReasonCodes)
	preview.PreviewDigest = digestUpgradePreview(preview)
	return preview
}

func noopUpgradePreview(preview UpgradePreview, snapshot ValidationSnapshot) UpgradePreview {
	preview.Status = upgradePreviewStatusNoop
	preview.ExpectedSnapshot = snapshot
	preview.PreviewDigest = digestUpgradePreview(preview)
	return preview
}

func readyUpgradePreview(preview UpgradePreview, expected ValidationSnapshot, before, after []byte) UpgradePreview {
	preview.Status = upgradePreviewStatusReady
	preview.ReasonCodes = []string{reasonUpgradeApplyExplicitRequired}
	preview.ExpectedSnapshot = expected
	preview.FileChanges = []UpgradeFileChange{{
		Path:             CanonicalConfigPath,
		Action:           "update",
		BeforeContentSHA: sha256Hex(before),
		AfterContentSHA:  sha256Hex(after),
	}}
	preview.RequiredFollowUp = []string{"review_preview", "apply_reviewed_upgrade", "revalidate_project_substrate"}
	preview.PreviewDigest = digestUpgradePreview(preview)
	return preview
}

func blockedUpgradePreviewDigestResult(preview UpgradePreview, expectedPreviewHash string) *UpgradeApplyResult {
	if strings.TrimSpace(expectedPreviewHash) == "" || strings.TrimSpace(expectedPreviewHash) == preview.PreviewDigest {
		return nil
	}
	result := blockedUpgradeApplyResult(preview.RepositoryRoot, preview.CurrentSnapshot, preview.PreviewDigest, []string{reasonUpgradePreviewDigestMismatch})
	return &result
}

func upgradePreviewAuthority(preview UpgradePreview) RepoRootAuthority {
	if strings.TrimSpace(preview.CurrentSnapshot.Contract.RepoRootAuthority) == string(RepoRootAuthorityProcessWorkingDirectory) {
		return RepoRootAuthorityProcessWorkingDirectory
	}
	return RepoRootAuthorityExplicitConfig
}

func noopUpgradeApplyResult(repoRoot string, snapshot ValidationSnapshot, previewDigest string) UpgradeApplyResult {
	return UpgradeApplyResult{
		SchemaID:          UpgradeApplySchemaID,
		SchemaVersion:     UpgradeApplyVersion,
		RepositoryRoot:    repoRoot,
		Status:            upgradeApplyStatusNoop,
		CurrentSnapshot:   snapshot,
		ResultingSnapshot: snapshot,
		PreviewDigest:     previewDigest,
	}
}

func blockedUpgradeApplyResult(repoRoot string, snapshot ValidationSnapshot, previewDigest string, reasonCodes []string) UpgradeApplyResult {
	blockedReasons := normalizeReasonCodes(append([]string{}, reasonCodes...))
	if len(blockedReasons) == 0 {
		blockedReasons = []string{reasonRemediationFlowRequired}
	}
	return UpgradeApplyResult{
		SchemaID:          UpgradeApplySchemaID,
		SchemaVersion:     UpgradeApplyVersion,
		RepositoryRoot:    repoRoot,
		Status:            upgradeApplyStatusBlocked,
		ReasonCodes:       blockedReasons,
		CurrentSnapshot:   snapshot,
		ResultingSnapshot: snapshot,
		PreviewDigest:     previewDigest,
	}
}

func appliedUpgradeResult(repoRoot string, current, resulting ValidationSnapshot, preview UpgradePreview) UpgradeApplyResult {
	return UpgradeApplyResult{
		SchemaID:          UpgradeApplySchemaID,
		SchemaVersion:     UpgradeApplyVersion,
		RepositoryRoot:    repoRoot,
		Status:            upgradeApplyStatusApplied,
		AppliedChanges:    append([]UpgradeFileChange{}, preview.FileChanges...),
		CurrentSnapshot:   current,
		ResultingSnapshot: resulting,
		PreviewDigest:     preview.PreviewDigest,
	}
}

func appendUpgradeAuditEvent(appender AuditEventAppender, repoRoot string, current ValidationSnapshot, applyResult UpgradeApplyResult) {
	if appender == nil {
		return
	}
	details := map[string]interface{}{
		"repository_root":           repoRoot,
		"preview_digest":            applyResult.PreviewDigest,
		"status":                    applyResult.Status,
		"reason_codes":              append([]string{}, applyResult.ReasonCodes...),
		"before_snapshot_digest":    current.SnapshotDigest,
		"result_snapshot_digest":    applyResult.ResultingSnapshot.SnapshotDigest,
		"validated_snapshot_digest": applyResult.ResultingSnapshot.ValidatedSnapshotDigest,
	}
	_ = appender.AppendTrustedAuditEvent("project_substrate_upgrade_event", "projectsubstrate", details)
}
