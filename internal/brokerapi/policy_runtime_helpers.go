package brokerapi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func allowlistIdentities(digests []trustpolicy.Digest) ([]string, error) {
	out := make([]string, 0, len(digests))
	for _, digest := range digests {
		identity, err := digest.Identity()
		if err != nil {
			return nil, err
		}
		out = append(out, identity)
	}
	return out, nil
}

func sortedUniquePolicyRefs(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	set := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func pickScopedRecord(records []artifacts.ArtifactRecord, runID string) (artifacts.ArtifactRecord, error) {
	if rec := pickOptionalRunRecord(records, runID); rec != nil {
		return *rec, nil
	}
	return artifacts.ArtifactRecord{}, fmt.Errorf("%w: no trusted import for run %q", errPolicyContextUnavailable, runID)
}

func pickRequiredRunRecord(records []artifacts.ArtifactRecord, runID string, kind string) (artifacts.ArtifactRecord, error) {
	rec := pickOptionalRunRecord(records, runID)
	if rec != nil {
		return *rec, nil
	}
	return artifacts.ArtifactRecord{}, fmt.Errorf("%w: no trusted %s for run %q", errPolicyContextUnavailable, kind, runID)
}

func pickOptionalRunRecord(records []artifacts.ArtifactRecord, runID string) *artifacts.ArtifactRecord {
	for i := range records {
		if strings.TrimSpace(records[i].RunID) == runID {
			return &records[i]
		}
	}
	for i := range records {
		if strings.TrimSpace(records[i].RunID) == "" {
			return &records[i]
		}
	}
	return nil
}

func digestFromIdentity(identity string) (trustpolicy.Digest, error) {
	if !strings.HasPrefix(identity, "sha256:") || len(identity) != 71 {
		return trustpolicy.Digest{}, fmt.Errorf("invalid digest identity %q", identity)
	}
	digest := trustpolicy.Digest{HashAlg: "sha256", Hash: strings.TrimPrefix(identity, "sha256:")}
	if _, err := digest.Identity(); err != nil {
		return trustpolicy.Digest{}, err
	}
	return digest, nil
}
