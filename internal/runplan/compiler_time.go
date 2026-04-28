package runplan

import "time"

func compiledAtRFC3339(compiledAt time.Time) string {
	resolved := compiledAt
	if resolved.IsZero() {
		resolved = time.Now().UTC()
	}
	return resolved.UTC().Format(time.RFC3339)
}
