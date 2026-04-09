package artifacts

import "strings"

func approvalHasBindingKeys(record *ApprovalRecord) bool {
	return strings.TrimSpace(record.ManifestHash) != "" && strings.TrimSpace(record.ActionRequestHash) != ""
}

func validOrEmptyPolicyDecisionHash(current string, decisions map[string]PolicyDecisionRecord) string {
	if strings.TrimSpace(current) == "" {
		return ""
	}
	if _, ok := decisions[current]; ok {
		return current
	}
	return ""
}

func matchingPolicyDecisionDigests(decisions map[string]PolicyDecisionRecord, manifestHash, actionRequestHash string) []string {
	matches := make([]string, 0, 2)
	for digest, decision := range decisions {
		if decision.ManifestHash == manifestHash && decision.ActionRequestHash == actionRequestHash {
			matches = append(matches, digest)
		}
	}
	return matches
}

func resolveApprovalPolicyDecisionHash(current string, matches []string, decisions map[string]PolicyDecisionRecord) (string, bool) {
	if len(matches) == 1 {
		return matches[0], true
	}
	if strings.TrimSpace(current) == "" {
		return "", len(matches) == 0
	}
	if _, ok := decisions[current]; !ok {
		return "", false
	}
	if len(matches) == 0 {
		return current, true
	}
	for _, digest := range matches {
		if digest == current {
			return current, true
		}
	}
	return "", false
}
