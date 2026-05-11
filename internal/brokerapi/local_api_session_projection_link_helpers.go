package brokerapi

import (
	"strings"

	"github.com/runecode-systems/runecode/internal/artifacts"
)

func sessionDetailLinkedRunIndex(base map[string]struct{}, executions []artifacts.SessionTurnExecutionDurableState) map[string]struct{} {
	out := copyLinkIndex(base)
	for _, execution := range executions {
		appendLinkIndexValue(out, execution.PrimaryRunID)
		appendLinkIndexValues(out, execution.LinkedRunIDs)
	}
	return out
}

func sessionDetailLinkedApprovalIndex(base map[string]struct{}, executions []artifacts.SessionTurnExecutionDurableState) map[string]struct{} {
	out := copyLinkIndex(base)
	for _, execution := range executions {
		appendLinkIndexValue(out, execution.PendingApprovalID)
		appendLinkIndexValues(out, execution.LinkedApprovalIDs)
	}
	return out
}

func sessionDetailLinkedArtifactIndex(base map[string]struct{}, executions []artifacts.SessionTurnExecutionDurableState) map[string]struct{} {
	return mergeExecutionLinkDigests(base, executions, func(execution artifacts.SessionTurnExecutionDurableState) []string {
		return execution.LinkedArtifactDigests
	})
}

func sessionDetailLinkedAuditIndex(base map[string]struct{}, executions []artifacts.SessionTurnExecutionDurableState) map[string]struct{} {
	return mergeExecutionLinkDigests(base, executions, func(execution artifacts.SessionTurnExecutionDurableState) []string {
		return execution.LinkedAuditRecordDigests
	})
}

func mergeExecutionLinkDigests(base map[string]struct{}, executions []artifacts.SessionTurnExecutionDurableState, selector func(artifacts.SessionTurnExecutionDurableState) []string) map[string]struct{} {
	out := copyLinkIndex(base)
	for _, execution := range executions {
		appendLinkIndexValues(out, selector(execution))
	}
	return out
}

func copyLinkIndex(in map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	for value := range in {
		out[value] = struct{}{}
	}
	return out
}

func appendLinkIndexValues(index map[string]struct{}, values []string) {
	for _, value := range values {
		appendLinkIndexValue(index, value)
	}
}

func appendLinkIndexValue(index map[string]struct{}, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	index[trimmed] = struct{}{}
}
