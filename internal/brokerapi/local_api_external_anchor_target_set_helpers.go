package brokerapi

import (
	"fmt"
	"sort"
	"strings"
)

func externalAnchorTypedRequestTargetSetEntries(typedRequest map[string]any) ([]any, error) {
	raw, hasSet := typedRequest["target_set"]
	if !hasSet || raw == nil {
		return nil, nil
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("typed_request.target_set invalid")
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("typed_request.target_set must include at least one target")
	}
	return entries, nil
}

func externalAnchorResolvedTargetIdentityKey(target externalAnchorResolvedTarget) string {
	return strings.TrimSpace(target.TargetKind) + "|" + strings.TrimSpace(target.TargetDescriptorIdentity)
}

func sortExternalAnchorResolvedTargets(targets []externalAnchorResolvedTarget) []externalAnchorResolvedTarget {
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].TargetKind != targets[j].TargetKind {
			return targets[i].TargetKind < targets[j].TargetKind
		}
		return targets[i].TargetDescriptorIdentity < targets[j].TargetDescriptorIdentity
	})
	return targets
}
