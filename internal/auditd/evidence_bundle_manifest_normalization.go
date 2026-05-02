package auditd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func evidenceBundleID(now time.Time) string {
	suffix := "0000000000000000"
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err == nil {
		suffix = hex.EncodeToString(randomBytes)
	}
	return "bundle-" + now.UTC().Format("20060102T150405Z") + "-" + suffix
}

func digestIdentityForVerifierRecord(record trustpolicy.VerifierRecord) (string, error) {
	keyValue := strings.TrimSpace(record.KeyIDValue)
	if keyValue == "" {
		return "", nil
	}
	identity := "sha256:" + keyValue
	if _, err := digestFromIdentity(identity); err != nil {
		return "", err
	}
	return identity, nil
}

func (l *Ledger) bundleObjectAbsolutePathLocked(relPath string) (string, error) {
	cleanRel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(relPath)))
	if cleanRel == "." || cleanRel == "" {
		return "", fmt.Errorf("bundle object path is required")
	}
	if filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) || cleanRel == ".." {
		return "", fmt.Errorf("bundle object path %q escapes ledger root", relPath)
	}
	root := filepath.Clean(l.rootDir)
	abs := filepath.Join(root, cleanRel)
	if abs != root && !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		return "", fmt.Errorf("bundle object path %q escapes ledger root", relPath)
	}
	return abs, nil
}

func evidenceBundleSidecarObjectPath(dirName string, digestIdentity string) string {
	return filepath.ToSlash(filepath.Join(sidecarDirName, dirName, strings.TrimPrefix(digestIdentity, "sha256:")+".json"))
}

func normalizeEvidenceBundleRedactions(redactions []AuditEvidenceBundleRedaction) []AuditEvidenceBundleRedaction {
	if len(redactions) == 0 {
		return nil
	}
	set := map[string]AuditEvidenceBundleRedaction{}
	for i := range redactions {
		path := strings.TrimSpace(redactions[i].Path)
		reason := strings.TrimSpace(redactions[i].ReasonCode)
		if path != "" && reason != "" {
			set[path+"|"+reason] = AuditEvidenceBundleRedaction{Path: path, ReasonCode: reason}
		}
	}
	out := make([]AuditEvidenceBundleRedaction, 0, len(set))
	for _, redaction := range set {
		out = append(out, redaction)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].ReasonCode < out[j].ReasonCode
		}
		return out[i].Path < out[j].Path
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeEvidenceBundleToolIdentity(identity AuditEvidenceBundleToolIdentity) AuditEvidenceBundleToolIdentity {
	return AuditEvidenceBundleToolIdentity{
		ToolName:                   strings.TrimSpace(identity.ToolName),
		ToolVersion:                strings.TrimSpace(identity.ToolVersion),
		BuildRevision:              strings.TrimSpace(identity.BuildRevision),
		ProtocolBundleManifestHash: strings.TrimSpace(identity.ProtocolBundleManifestHash),
	}
}

func normalizeEvidenceBundleScope(scope AuditEvidenceBundleScope) AuditEvidenceBundleScope {
	artifactDigests := normalizeIdentityList(scope.ArtifactDigests)
	if len(artifactDigests) == 0 {
		artifactDigests = nil
	}
	return AuditEvidenceBundleScope{
		ScopeKind:       strings.TrimSpace(scope.ScopeKind),
		RunID:           strings.TrimSpace(scope.RunID),
		IncidentID:      strings.TrimSpace(scope.IncidentID),
		ArtifactDigests: artifactDigests,
	}
}

func normalizeEvidenceBundleDisclosurePosture(posture AuditEvidenceBundleDisclosurePosture) AuditEvidenceBundleDisclosurePosture {
	return AuditEvidenceBundleDisclosurePosture{
		Posture:                    strings.TrimSpace(posture.Posture),
		SelectiveDisclosureApplied: posture.SelectiveDisclosureApplied,
	}
}
