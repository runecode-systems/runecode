package auditd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) rebuildSealLookupsLocked(idx *proofLookupIndex) error {
	entries, err := l.readOptionalSidecarDirEntries(sealsDirName)
	if err != nil {
		return err
	}
	for i := range entries {
		digest, payload, ok, err := l.loadVerifiedSegmentSealCandidate(entries[i])
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err := idx.notePersistedSegmentSeal(digest, payload); err != nil {
			return err
		}
	}
	return nil
}

func (l *Ledger) loadVerifiedSegmentSealCandidate(entry os.DirEntry) (trustpolicy.Digest, trustpolicy.AuditSegmentSealPayload, bool, error) {
	if entry.IsDir() {
		return trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, false, nil
	}
	identity, ok, err := digestIdentityFromSidecarName(entry.Name())
	if err != nil || !ok {
		return trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, ok, err
	}
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, false, err
	}
	payload, err := l.readVerifiedSegmentSealPayload(identity)
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.AuditSegmentSealPayload{}, false, err
	}
	return digest, payload, true, nil
}

func (l *Ledger) readVerifiedSegmentSealPayload(identity string) (trustpolicy.AuditSegmentSealPayload, error) {
	envelope := trustpolicy.SignedObjectEnvelope{}
	if err := readJSONFile(sidecarPath(l.rootDir, sealsDirName, identity), &envelope); err != nil {
		return trustpolicy.AuditSegmentSealPayload{}, err
	}
	computedDigest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return trustpolicy.AuditSegmentSealPayload{}, err
	}
	if mustDigestIdentity(computedDigest) != identity {
		return trustpolicy.AuditSegmentSealPayload{}, fmt.Errorf("segment seal content digest mismatch for %s", identity)
	}
	return decodeAndValidateSealEnvelope(envelope)
}

func (idx *proofLookupIndex) notePersistedSegmentSeal(sealDigest trustpolicy.Digest, payload trustpolicy.AuditSegmentSealPayload) error {
	if idx == nil {
		return fmt.Errorf("proof lookup index is required")
	}
	normalizeProofLookupIndex(idx)
	identity, err := sealDigest.Identity()
	if err != nil {
		return err
	}
	segmentID := strings.TrimSpace(payload.SegmentID)
	candidate := segmentSealLookup{DigestIdentity: identity, SealChainIndex: payload.SealChainIndex}
	current, ok := idx.SegmentSeals[segmentID]
	if !ok || candidate.SealChainIndex > current.SealChainIndex || (candidate.SealChainIndex == current.SealChainIndex && candidate.DigestIdentity > current.DigestIdentity) {
		idx.SegmentSeals[segmentID] = candidate
	}
	chainKey := strconv.FormatInt(payload.SealChainIndex, 10)
	if cur, ok := idx.SealByChainIndex[chainKey]; !ok || identity > cur {
		idx.SealByChainIndex[chainKey] = identity
	}
	return nil
}
