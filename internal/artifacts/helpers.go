package artifacts

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

const backupHMACKeyEnv = "RUNE_BACKUP_HMAC_KEY"

func defaultBlobDir(rootDir string) string {
	return filepath.Join(rootDir, "blobs")
}

func isValidDigest(digest string) bool {
	return digestPattern.MatchString(digest)
}

func digestBytes(payload []byte) string {
	h := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(h[:])
}

func DigestBytes(payload []byte) string {
	return digestBytes(payload)
}

func canonicalPayload(contentType string, payload []byte) ([]byte, error) {
	if isJSONContentType(contentType) {
		canonical, err := canonicalizeJSONBytes(payload)
		if err != nil {
			return nil, fmt.Errorf("invalid json payload: %w", err)
		}
		return canonical, nil
	}
	return append([]byte(nil), payload...), nil
}

func isReservedDataClass(dataClass DataClass) bool {
	return dataClass == DataClassWebQuery || dataClass == DataClassWebCitations
}

func normalizeState(state StoreState) StoreState {
	state = normalizePrimaryStateMaps(state)
	state = normalizeRuntimeStateMaps(state)
	if state.Policy.HandOffReferenceMode == "" {
		state.Policy = DefaultPolicy()
	}
	if state.StorageProtectionPosture == "" {
		state.StorageProtectionPosture = "encrypted_at_rest_default"
	}
	return state
}

func normalizePrimaryStateMaps(state StoreState) StoreState {
	state = normalizeCoreStateMaps(state)
	state = normalizeApprovalStateMaps(state)
	state = normalizeProviderStateMaps(state)
	return normalizePromotionStateMaps(state)
}

func normalizeCoreStateMaps(state StoreState) StoreState {
	if state.Artifacts == nil {
		state.Artifacts = map[string]ArtifactRecord{}
	}
	if state.Sessions == nil {
		state.Sessions = map[string]SessionDurableState{}
	}
	if state.Runs == nil {
		state.Runs = map[string]string{}
	}
	if state.PolicyDecisions == nil {
		state.PolicyDecisions = map[string]PolicyDecisionRecord{}
	}
	if state.RunPolicyDecisionRefs == nil {
		state.RunPolicyDecisionRefs = map[string][]string{}
	}
	return state
}

func normalizeApprovalStateMaps(state StoreState) StoreState {
	if state.Approvals == nil {
		state.Approvals = map[string]ApprovalRecord{}
	}
	if state.RunApprovalRefs == nil {
		state.RunApprovalRefs = map[string][]string{}
	}
	if state.GitRemotePrepared == nil {
		state.GitRemotePrepared = map[string]GitRemotePreparedMutationRecord{}
	}
	if state.RunGitRemotePreparedRefs == nil {
		state.RunGitRemotePreparedRefs = map[string][]string{}
	}
	return state
}

func normalizeProviderStateMaps(state StoreState) StoreState {
	if state.ProviderProfiles == nil {
		state.ProviderProfiles = map[string]ProviderProfileDurableState{}
	}
	if state.ProviderSetupSessions == nil {
		state.ProviderSetupSessions = map[string]ProviderSetupSessionDurableState{}
	}
	return state
}

func normalizePromotionStateMaps(state StoreState) StoreState {
	if state.PromotionEventsByActor == nil {
		state.PromotionEventsByActor = map[string][]time.Time{}
	}
	return state
}

func normalizeRuntimeStateMaps(state StoreState) StoreState {
	if state.RuntimeFactsByRun == nil {
		state.RuntimeFactsByRun = map[string]launcherbackend.RuntimeFactsSnapshot{}
	}
	if state.RuntimeEvidenceByRun == nil {
		state.RuntimeEvidenceByRun = map[string]launcherbackend.RuntimeEvidenceSnapshot{}
	}
	if state.RuntimeLifecycleByRun == nil {
		state.RuntimeLifecycleByRun = map[string]launcherbackend.RuntimeLifecycleState{}
	}
	if state.RuntimeAuditStateByRun == nil {
		state.RuntimeAuditStateByRun = map[string]RuntimeAuditEmissionState{}
	}
	if state.RunnerAdvisoryByRun == nil {
		state.RunnerAdvisoryByRun = map[string]RunnerAdvisoryState{}
	}
	return state
}

func ensureBackupKey(state StoreState) (StoreState, error) {
	if state.BackupHMACKey != "" {
		return state, nil
	}
	if configured := os.Getenv(backupHMACKeyEnv); configured != "" {
		state.BackupHMACKey = configured
		return state, nil
	}
	randomKey, err := randomBackupKey()
	if err != nil {
		return state, err
	}
	state.BackupHMACKey = randomKey
	return state, nil
}

func buildArtifactReference(digest string, size int64, req PutRequest) ArtifactReference {
	return ArtifactReference{
		Digest:                digest,
		SizeBytes:             size,
		ContentType:           req.ContentType,
		DataClass:             req.DataClass,
		ProvenanceReceiptHash: req.ProvenanceReceiptHash,
	}
}

func validateRunStatusInput(runID, status string) error {
	if runID == "" {
		return fmt.Errorf("run id is required")
	}
	if !isSupportedRunStatus(status) {
		return fmt.Errorf("unsupported run status")
	}
	return nil
}

func isSupportedRunStatus(status string) bool {
	switch status {
	case "pending", "starting", "active", "blocked", "recovering", "completed", "failed", "cancelled", "retained", "closed":
		return true
	default:
		return false
	}
}

func backupSignaturePath(manifestPath string) string {
	return manifestPath + ".sig"
}

func computeBackupSignature(manifest BackupManifest, key string) (BackupSignature, error) {
	canonical, err := canonicalBackupManifestBytes(manifest)
	if err != nil {
		return BackupSignature{}, err
	}
	manifestHash := sha256.Sum256(canonical)
	h := hmac.New(sha256.New, []byte(key))
	if _, err := h.Write(canonical); err != nil {
		return BackupSignature{}, err
	}
	mac := h.Sum(nil)
	return BackupSignature{
		Schema:         "runecode.backup.signature.v1",
		ManifestSHA256: hex.EncodeToString(manifestHash[:]),
		HMACSHA256:     hex.EncodeToString(mac),
		KeyID:          backupKeyID(key),
		ExportedAt:     manifest.ExportedAt,
	}, nil
}

func verifyBackupSignature(manifest BackupManifest, signature BackupSignature, key string) error {
	if signature.Schema != "runecode.backup.signature.v1" {
		return ErrBackupSignatureInvalid
	}
	if signature.KeyID != backupKeyID(key) {
		return ErrBackupSignatureInvalid
	}
	expected, err := computeBackupSignature(manifest, key)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(expected.ManifestSHA256), []byte(signature.ManifestSHA256)) {
		return ErrBackupSignatureInvalid
	}
	if !hmac.Equal([]byte(expected.HMACSHA256), []byte(signature.HMACSHA256)) {
		return ErrBackupSignatureInvalid
	}
	return nil
}

func canonicalBackupManifestBytes(manifest BackupManifest) ([]byte, error) {
	b, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	return canonicalizeJSONBytes(b)
}

func backupKeyID(key string) string {
	h := sha256.Sum256([]byte(key))
	return "sha256:" + hex.EncodeToString(h[:16])
}

func sanitizeBackupPath(filePath string) string {
	return filepath.Clean(filePath)
}

func randomBackupKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "hmac-sha256:" + hex.EncodeToString(b), nil
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	clone := append([]string{}, values...)
	sort.Strings(clone)
	out := make([]string, 0, len(clone))
	for _, v := range clone {
		if len(out) == 0 || out[len(out)-1] != v {
			out = append(out, v)
		}
	}
	return out
}
