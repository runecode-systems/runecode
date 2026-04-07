package brokerapi

import "sort"

func sortArtifactSummariesNewestFirst(items []ArtifactSummary) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].Reference.Digest > items[j].Reference.Digest
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
}
