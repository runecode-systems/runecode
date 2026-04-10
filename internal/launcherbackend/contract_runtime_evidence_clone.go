package launcherbackend

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for k, v := range values {
		out[k] = v
	}
	return out
}

func cloneQEMUProvenance(value *QEMUProvenance) *QEMUProvenance {
	if value == nil {
		return nil
	}
	trimmed := value.Trimmed()
	return &trimmed
}

func cloneCachePosture(value *BackendCachePosture) *BackendCachePosture {
	if value == nil {
		return nil
	}
	out := *value
	return &out
}

func cloneCacheEvidence(value *BackendCacheEvidence) *BackendCacheEvidence {
	if value == nil {
		return nil
	}
	out := *value
	out.ResolvedBootComponentDigests = uniqueSortedStrings(value.ResolvedBootComponentDigests)
	return &out
}

func cloneLifecycle(value *BackendLifecycleSnapshot) *BackendLifecycleSnapshot {
	if value == nil {
		return nil
	}
	out := value.Normalized()
	return &out
}

func cloneWorkspaceEncryptionPosture(value *WorkspaceEncryptionPosture) *WorkspaceEncryptionPosture {
	if value == nil {
		return nil
	}
	normalized := value.Normalized()
	return &normalized
}

func cloneAttachmentPlanSummary(value *AttachmentPlanSummary) *AttachmentPlanSummary {
	if value == nil {
		return nil
	}
	normalized := value.Normalized()
	return &normalized
}

func cloneSessionSecurityPosture(value *SessionSecurityPosture) *SessionSecurityPosture {
	if value == nil {
		return nil
	}
	out := *value
	out.DegradedReasons = uniqueSortedStrings(value.DegradedReasons)
	return &out
}
