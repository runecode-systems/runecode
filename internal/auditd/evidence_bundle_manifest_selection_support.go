package auditd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type bundleManifestReceiptPayload struct {
	SubjectDigest trustpolicy.Digest `json:"subject_digest"`
}

func (l *Ledger) shouldIncludeReceiptForSegmentSetLocked(receiptDigestIdentity string, segmentSet map[string]struct{}) (bool, error) {
	envelope, err := l.loadReceiptEnvelopeByDigestIdentityLocked(receiptDigestIdentity)
	if err != nil {
		return false, err
	}
	payload := bundleManifestReceiptPayload{}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return false, err
	}
	subjectIdentity, err := payload.SubjectDigest.Identity()
	if err != nil {
		return false, err
	}
	for segmentID := range segmentSet {
		lookup, ok, lookupErr := l.lookupSegmentSealLocked(segmentID, false)
		if lookupErr != nil {
			return false, lookupErr
		}
		if ok && subjectIdentity == strings.TrimSpace(lookup.SealDigest) {
			return true, nil
		}
	}
	return false, nil
}

func (l *Ledger) loadReceiptEnvelopeByDigestIdentityLocked(identity string) (trustpolicy.SignedObjectEnvelope, error) {
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	envelope := trustpolicy.SignedObjectEnvelope{}
	path := filepath.Join(l.rootDir, sidecarDirName, receiptsDirName, digest.Hash+".json")
	if err := readJSONFile(path, &envelope); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	computed, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	computedIdentity, _ := computed.Identity()
	if computedIdentity != identity {
		return trustpolicy.SignedObjectEnvelope{}, fmt.Errorf("receipt envelope digest mismatch: expected %q computed %q", identity, computedIdentity)
	}
	return envelope, nil
}

func (l *Ledger) shouldIncludeVerificationReportForSegmentSetLocked(reportDigestIdentity string, segmentSet map[string]struct{}) (bool, error) {
	report, err := l.loadVerificationReportByDigestIdentityLocked(reportDigestIdentity)
	if err != nil {
		return false, err
	}
	lastSegmentID := strings.TrimSpace(report.VerificationScope.LastSegmentID)
	if lastSegmentID == "" {
		return false, nil
	}
	_, ok := segmentSet[lastSegmentID]
	return ok, nil
}

func (l *Ledger) evidenceBundleObjectByteLengthLocked(relPath string) (int64, error) {
	segmentID, ok := segmentIDFromPath(relPath)
	if ok {
		segment, err := l.loadSegment(segmentID)
		if err != nil {
			return 0, err
		}
		raw, err := l.rawSegmentFramedBytes(segment)
		if err != nil {
			return 0, err
		}
		return int64(len(raw)), nil
	}
	absolute, err := l.bundleObjectAbsolutePathLocked(relPath)
	if err != nil {
		return 0, err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func segmentIDFromPath(relPath string) (string, bool) {
	if !strings.HasPrefix(relPath, segmentsDirName+"/") || !strings.HasSuffix(relPath, ".json") {
		return "", false
	}
	name := strings.TrimPrefix(relPath, segmentsDirName+"/")
	return strings.TrimSuffix(name, ".json"), true
}

func (l *Ledger) evidenceBundleRootsAndSealsLocked(segmentIDs []string) ([]string, []AuditEvidenceBundleSealReference, error) {
	rootSet := map[string]struct{}{}
	sealRefs := make([]AuditEvidenceBundleSealReference, 0, len(segmentIDs))
	for i := range segmentIDs {
		_, digest, seal, err := l.loadSealEnvelopeForSegmentLocked(segmentIDs[i])
		if err != nil {
			return nil, nil, err
		}
		addEvidenceBundleRootIdentities(rootSet, digest, seal)
		sealRefs = append(sealRefs, AuditEvidenceBundleSealReference{SegmentID: seal.SegmentID, SealDigest: digestIdentityOrEmpty(digest), SealChainIndex: seal.SealChainIndex, PreviousSealDigest: previousSealDigestIdentity(seal)})
	}
	return sortEvidenceBundleRoots(rootSet), sortEvidenceBundleSealReferences(sealRefs), nil
}

func addEvidenceBundleRootIdentities(rootSet map[string]struct{}, digest trustpolicy.Digest, seal trustpolicy.AuditSegmentSealPayload) {
	addNonEmptyRootIdentity(rootSet, digestIdentityOrEmpty(digest))
	addNonEmptyRootIdentity(rootSet, digestPointerIdentityOrEmpty(&seal.MerkleRoot))
	addNonEmptyRootIdentity(rootSet, digestPointerIdentityOrEmpty(&seal.SegmentFileHash))
}

func addNonEmptyRootIdentity(rootSet map[string]struct{}, identity string) {
	if identity != "" {
		rootSet[identity] = struct{}{}
	}
}

func digestIdentityOrEmpty(digest trustpolicy.Digest) string {
	identity, _ := digest.Identity()
	return identity
}

func digestPointerIdentityOrEmpty(digest *trustpolicy.Digest) string {
	if digest == nil {
		return ""
	}
	identity, _ := digest.Identity()
	return identity
}

func sortEvidenceBundleRoots(rootSet map[string]struct{}) []string {
	rootDigests := make([]string, 0, len(rootSet))
	for digest := range rootSet {
		rootDigests = append(rootDigests, digest)
	}
	sort.Strings(rootDigests)
	return rootDigests
}

func sortEvidenceBundleSealReferences(sealRefs []AuditEvidenceBundleSealReference) []AuditEvidenceBundleSealReference {
	sort.Slice(sealRefs, func(i, j int) bool {
		if sealRefs[i].SealChainIndex == sealRefs[j].SealChainIndex {
			return sealRefs[i].SegmentID < sealRefs[j].SegmentID
		}
		return sealRefs[i].SealChainIndex < sealRefs[j].SealChainIndex
	})
	return sealRefs
}
