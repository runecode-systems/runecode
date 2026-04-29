package launcherdaemon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func validateRuntimeVerifierAuthorityStateDigest(state runtimeVerifierAuthorityState) error {
	if strings.TrimSpace(state.StateDigest) != mustRuntimeVerifierAuthorityStateDigest(state) {
		return fmt.Errorf("runtime verifier authority state digest mismatch")
	}
	return nil
}

func validateRuntimeVerifierAuthorityEntries(authoritiesByKind map[string][]runtimeVerifierAuthorityEntry) error {
	for kind, entries := range authoritiesByKind {
		if strings.TrimSpace(kind) == "" {
			return fmt.Errorf("runtime verifier authority state kind is empty")
		}
		if err := validateRuntimeVerifierAuthorityEntriesForKind(entries); err != nil {
			return err
		}
	}
	return nil
}

func validateRuntimeVerifierAuthorityEntriesForKind(entries []runtimeVerifierAuthorityEntry) error {
	seenVerifierSets := map[string]struct{}{}
	for _, entry := range entries {
		if err := validateRuntimeVerifierAuthorityEntry(entry, seenVerifierSets); err != nil {
			return err
		}
	}
	return nil
}

func validateRuntimeVerifierAuthorityEntry(entry runtimeVerifierAuthorityEntry, seenVerifierSets map[string]struct{}) error {
	if !isDigestFormat(entry.VerifierSetRef) {
		return fmt.Errorf("runtime verifier authority entry verifier_set_ref is invalid")
	}
	if _, exists := seenVerifierSets[entry.VerifierSetRef]; exists {
		return fmt.Errorf("runtime verifier authority entry verifier_set_ref must be unique per kind")
	}
	seenVerifierSets[entry.VerifierSetRef] = struct{}{}
	if len(entry.Records) == 0 {
		return fmt.Errorf("runtime verifier authority entry records are required")
	}
	if _, err := trustpolicy.NewVerifierRegistry(entry.Records); err != nil {
		return fmt.Errorf("runtime verifier authority entry records are invalid: %w", err)
	}
	if entry.VerifierSetRef != mustRuntimeVerifierSetDigest(entry.Records) {
		return fmt.Errorf("runtime verifier authority entry digest does not match records")
	}
	if entry.Status != runtimeVerifierAuthorityStatusActive && entry.Status != runtimeVerifierAuthorityStatusRevoked {
		return fmt.Errorf("runtime verifier authority entry status is invalid")
	}
	if entry.Source != runtimeVerifierAuthoritySourceBuiltin && entry.Source != runtimeVerifierAuthoritySourceImported {
		return fmt.Errorf("runtime verifier authority entry source is invalid")
	}
	if _, err := time.Parse(time.RFC3339, entry.ChangedAt); err != nil {
		return fmt.Errorf("runtime verifier authority entry changed_at is invalid")
	}
	return nil
}

func validateRuntimeVerifierAuthorityStateProgression(next runtimeVerifierAuthorityState, previousEffective runtimeVerifierAuthorityState, previousImported runtimeVerifierAuthorityState, foundImported bool) error {
	if isIdempotentRuntimeVerifierAuthorityImport(next, previousImported, foundImported) {
		return nil
	}
	previousRevision := previousEffective.Generation.Revision
	if foundImported {
		previousRevision = previousImported.Generation.Revision
	}
	if next.Generation.Revision <= previousRevision {
		return fmt.Errorf("runtime verifier authority state generation.revision must advance monotonically")
	}
	if next.Generation.PreviousRevision != previousRevision {
		return fmt.Errorf("runtime verifier authority state generation.previous_revision must match current revision")
	}
	return nil
}

func isIdempotentRuntimeVerifierAuthorityImport(next runtimeVerifierAuthorityState, previousImported runtimeVerifierAuthorityState, foundImported bool) bool {
	if !foundImported {
		return false
	}
	return next.Generation.Revision == previousImported.Generation.Revision && next.StateDigest == previousImported.StateDigest
}

func mustRuntimeVerifierAuthorityStateDigest(state runtimeVerifierAuthorityState) string {
	copy := state
	copy.StateDigest = ""
	b, err := json.Marshal(copy)
	if err != nil {
		panic(err)
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}
