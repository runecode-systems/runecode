package projectsubstrate

func AdoptExisting(input AdoptionInput) (AdoptionResult, error) {
	result, err := DiscoverAndValidate(DiscoveryInput{RepositoryRoot: input.RepositoryRoot, Authority: input.Authority})
	if err != nil {
		return AdoptionResult{}, err
	}
	out := AdoptionResult{
		SchemaID:       AdoptionSchemaID,
		SchemaVersion:  AdoptionSchemaVersion,
		RepositoryRoot: result.RepositoryRoot,
		Snapshot:       result.Snapshot,
	}
	if AllowsNormalOperation(result.Compatibility.Posture) {
		out.Status = adoptionStatusAdopted
		return out, nil
	}
	out.Status = adoptionStatusBlocked
	out.ReasonCodes = normalizeReasonCodes(result.Compatibility.BlockedReasonCodes)
	return out, nil
}

func PreviewInitialize(input InitPreviewInput) (InitPreview, error) {
	discovered, layout, preview, err := previewInitializeBase(input)
	if err != nil {
		return InitPreview{}, err
	}
	if preview, ok := noopInitializePreview(discovered.Snapshot, preview); ok {
		return preview, nil
	}
	if preview, ok := blockedInitializePreview(discovered.Snapshot, preview); ok {
		return preview, nil
	}
	return readyInitializePreview(discovered, layout, preview), nil
}

func ApplyInitialize(input InitApplyInput) (InitApplyResult, error) {
	preview := input.Preview
	if mismatch := blockedInitPreviewTokenResult(preview, input.ExpectedPreviewToken); mismatch != nil {
		return *mismatch, nil
	}
	current, authority, err := initializeCurrentSnapshot(preview)
	if err != nil {
		return InitApplyResult{}, err
	}
	if current.Snapshot.SnapshotDigest != preview.CurrentSnapshot.SnapshotDigest {
		return blockedInitSnapshotResult(current.RepositoryRoot, current.Snapshot, preview.PreviewToken), nil
	}
	if result, ok := earlyInitApplyResult(preview, current); ok {
		return result, nil
	}
	result, err := applyInitializePreview(preview, current, authority)
	if err != nil {
		return InitApplyResult{}, err
	}
	if err := appendInitAuditEvent(input.AuditAppender, current.RepositoryRoot, current.Snapshot, result); err != nil {
		result.ReasonCodes = normalizeReasonCodes(append(result.ReasonCodes, reasonAuditAppendFailed))
	}
	return result, nil
}

func previewInitializeBase(input InitPreviewInput) (DiscoveryResult, repositoryLayout, InitPreview, error) {
	discovered, err := DiscoverAndValidate(DiscoveryInput{RepositoryRoot: input.RepositoryRoot, Authority: input.Authority})
	if err != nil {
		return DiscoveryResult{}, repositoryLayout{}, InitPreview{}, err
	}
	preview := InitPreview{SchemaID: InitPreviewSchemaID, SchemaVersion: InitPreviewVersion, RepositoryRoot: discovered.RepositoryRoot, CurrentSnapshot: discovered.Snapshot}
	return discovered, inspectLayout(discovered.RepositoryRoot), preview, nil
}

func noopInitializePreview(snapshot ValidationSnapshot, preview InitPreview) (InitPreview, bool) {
	if snapshot.ValidationState != validationStateValid {
		return InitPreview{}, false
	}
	preview.Status = initPreviewStatusNoop
	preview.ExpectedSnapshot = snapshot
	preview.PreviewToken = digestInitPreview(preview)
	return preview, true
}

func blockedInitializePreview(snapshot ValidationSnapshot, preview InitPreview) (InitPreview, bool) {
	conflictReasons, conflictPaths := initConflicts(snapshot)
	if len(conflictReasons) == 0 {
		return InitPreview{}, false
	}
	preview.Status = initPreviewStatusBlocked
	preview.ReasonCodes = normalizeReasonCodes(append(conflictReasons, reasonInitConflictDetected))
	preview.ExpectedSnapshot = snapshot
	preview.ConflictingPaths = conflictPaths
	preview.RequiredFollowUp = initRemediationFollowUp(preview.ReasonCodes)
	preview.PreviewToken = digestInitPreview(preview)
	return preview, true
}

func readyInitializePreview(discovered DiscoveryResult, layout repositoryLayout, preview InitPreview) InitPreview {
	mutation, err := canonicalInitialization(recommendedRuneContextVersionTarget(), "embedded")
	if err != nil {
		preview.Status = initPreviewStatusBlocked
		preview.ReasonCodes = []string{reasonRemediationFlowRequired}
		preview.ExpectedSnapshot = discovered.Snapshot
		preview.RequiredFollowUp = []string{"inspect_project_substrate_diagnostics"}
		preview.PreviewToken = digestInitPreview(preview)
		return preview
	}
	nextLayout := layout
	nextLayout.hasConfigAnchor = true
	nextLayout.hasSourceAnchor = true
	nextLayout.hasAssuranceAnchor = true
	nextLayout.hasAssuranceBaseline = true
	nextLayout.runecontextYAML = []byte(mutation.ConfigYAML)
	preview.Status = initPreviewStatusReady
	preview.ExpectedSnapshot = validateLayout(discovered.Contract, nextLayout)
	preview.FileChanges = []InitFileChange{
		{Path: mutation.SourcePath, Action: "create_directory"},
		{Path: mutation.AssurancePath, Action: "create_directory"},
		{Path: mutation.BaselinePath, Action: "create_file", AfterContentSHA: sha256Hex([]byte(mutation.AssuranceBaselineYML))},
		{Path: CanonicalConfigPath, Action: "create_file", AfterContentSHA: sha256Hex([]byte(mutation.ConfigYAML))},
	}
	preview.RequiredFollowUp = []string{"review_init_preview", "apply_init_preview", "revalidate_project_substrate"}
	preview.PreviewToken = digestInitPreview(preview)
	return preview
}

func initializeCurrentSnapshot(preview InitPreview) (DiscoveryResult, RepoRootAuthority, error) {
	authority := initPreviewAuthority(preview)
	current, err := DiscoverAndValidate(DiscoveryInput{RepositoryRoot: preview.RepositoryRoot, Authority: authority})
	return current, authority, err
}

func earlyInitApplyResult(preview InitPreview, current DiscoveryResult) (InitApplyResult, bool) {
	if preview.Status == initPreviewStatusNoop {
		return noopInitApplyResult(current.RepositoryRoot, current.Snapshot, preview.PreviewToken), true
	}
	if preview.Status != initPreviewStatusReady {
		return blockedInitApplyResult(current.RepositoryRoot, current.Snapshot, preview.PreviewToken, preview.ReasonCodes), true
	}
	return InitApplyResult{}, false
}

func applyInitializePreview(preview InitPreview, current DiscoveryResult, authority RepoRootAuthority) (InitApplyResult, error) {
	root := current.RepositoryRoot
	blockedReasons, _, err := initApplyConflicts(root)
	if err != nil {
		return InitApplyResult{}, err
	}
	if len(blockedReasons) > 0 {
		return blockedInitApplyResult(root, current.Snapshot, preview.PreviewToken, append(blockedReasons, reasonInitConflictDetected)), nil
	}
	if err := applyCanonicalInitialization(root); err != nil {
		return InitApplyResult{}, err
	}
	resultSnapshot, err := DiscoverAndValidate(DiscoveryInput{RepositoryRoot: root, Authority: authority})
	if err != nil {
		return InitApplyResult{}, err
	}
	result := InitApplyResult{SchemaID: InitApplySchemaID, SchemaVersion: InitApplyVersion, RepositoryRoot: root, Status: initApplyStatusApplied, AppliedChanges: append([]InitFileChange{}, preview.FileChanges...), CurrentSnapshot: current.Snapshot, ResultingSnapshot: resultSnapshot.Snapshot, PreviewToken: preview.PreviewToken}
	if result.ResultingSnapshot.ValidationState != validationStateValid {
		result.Status = initApplyStatusAppliedInvalid
		result.ReasonCodes = []string{reasonInitPostValidationFailed}
	}
	return result, nil
}
