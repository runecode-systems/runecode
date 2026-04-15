package auditd

import (
	"errors"
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

var ErrNoSealedSegment = errors.New("no sealed segment available")

// LatestAnchorableSeal returns the most recent sealed segment identity and seal digest.
func (l *Ledger) LatestAnchorableSeal() (string, trustpolicy.Digest, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	state, err := l.recoverAndPersistStateLocked()
	if err != nil {
		return "", trustpolicy.Digest{}, err
	}
	if state.LastSealedSegmentID == "" {
		return "", trustpolicy.Digest{}, ErrNoSealedSegment
	}
	_, digest, _, err := l.loadSealEnvelopeForSegmentLocked(state.LastSealedSegmentID)
	if err != nil {
		return "", trustpolicy.Digest{}, err
	}
	if _, err := digest.Identity(); err != nil {
		return "", trustpolicy.Digest{}, fmt.Errorf("latest seal digest invalid: %w", err)
	}
	return state.LastSealedSegmentID, digest, nil
}
