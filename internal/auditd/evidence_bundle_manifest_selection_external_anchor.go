package auditd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) shouldIncludeExternalAnchorObjectForSegmentSetLocked(identity string, family string, segmentSet map[string]struct{}) (bool, error) {
	evidence, digest, ok, err := l.loadExternalAnchorEvidenceByDigestIdentityLocked(identity)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	matched, err := l.externalAnchorMatchesSelectedSegmentLocked(evidence, segmentSet)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, nil
	}
	if family == "external_anchor_evidence" {
		return true, nil
	}
	return externalAnchorIncludesSidecarIdentity(evidence, digest, identity)
}

func (l *Ledger) externalAnchorMatchesSelectedSegmentLocked(evidence trustpolicy.ExternalAnchorEvidencePayload, segmentSet map[string]struct{}) (bool, error) {
	anchoringSubjectIdentity, err := evidence.AnchoringSubjectDigest.Identity()
	if err != nil {
		return false, err
	}
	for segmentID := range segmentSet {
		lookup, ok, err := l.lookupSegmentSealLocked(segmentID, false)
		if err != nil {
			return false, err
		}
		if ok && anchoringSubjectIdentity == strings.TrimSpace(lookup.SealDigest) {
			return true, nil
		}
	}
	return false, nil
}

func externalAnchorIncludesSidecarIdentity(evidence trustpolicy.ExternalAnchorEvidencePayload, digest *trustpolicy.Digest, identity string) (bool, error) {
	if digest == nil {
		return false, nil
	}
	for i := range evidence.SidecarRefs {
		refIdentity, err := evidence.SidecarRefs[i].Digest.Identity()
		if err != nil {
			return false, err
		}
		if refIdentity == identity {
			return true, nil
		}
	}
	return false, nil
}

func (l *Ledger) loadExternalAnchorEvidenceByDigestIdentityLocked(identity string) (trustpolicy.ExternalAnchorEvidencePayload, *trustpolicy.Digest, bool, error) {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, externalAnchorEvidenceDir))
	if err != nil {
		if os.IsNotExist(err) {
			return trustpolicy.ExternalAnchorEvidencePayload{}, nil, false, nil
		}
		return trustpolicy.ExternalAnchorEvidencePayload{}, nil, false, err
	}
	for _, entry := range entries {
		rec, digest, ok, readErr := l.readExternalAnchorEvidenceDirEntry(entry)
		if readErr != nil {
			return trustpolicy.ExternalAnchorEvidencePayload{}, nil, false, readErr
		}
		matched, matchErr := externalAnchorEvidenceMatchesIdentity(rec, digest, ok, identity)
		if matchErr != nil {
			return trustpolicy.ExternalAnchorEvidencePayload{}, nil, false, matchErr
		}
		if matched {
			return rec, digest, true, nil
		}
	}
	return trustpolicy.ExternalAnchorEvidencePayload{}, nil, false, nil
}

func externalAnchorEvidenceMatchesIdentity(rec trustpolicy.ExternalAnchorEvidencePayload, digest *trustpolicy.Digest, ok bool, identity string) (bool, error) {
	if !ok || digest == nil {
		return false, nil
	}
	digestIdentity, err := digest.Identity()
	if err != nil {
		return false, err
	}
	if digestIdentity == identity {
		return true, nil
	}
	for i := range rec.SidecarRefs {
		refIdentity, err := rec.SidecarRefs[i].Digest.Identity()
		if err != nil {
			return false, err
		}
		if refIdentity == identity {
			return true, nil
		}
	}
	return false, nil
}
