//go:build linux

package launcherdaemon

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func buildLaunchReceipt(spec launcherbackend.BackendLaunchSpec, admission launcherbackend.RuntimeAdmissionRecord, isoID, sessionID, nonce, qemuVersion, qemuBuild string, cacheEvidence *launcherbackend.BackendCacheEvidence, now time.Time) (launcherbackend.BackendLaunchReceipt, error) {
	receipt := launcherbackend.BackendLaunchReceipt{
		RunID:                    spec.RunID,
		StageID:                  spec.StageID,
		RoleInstanceID:           spec.RoleInstanceID,
		RoleFamily:               spec.RoleFamily,
		RoleKind:                 spec.RoleKind,
		BackendKind:              launcherbackend.BackendKindMicroVM,
		IsolationAssuranceLevel:  launcherbackend.IsolationAssuranceIsolated,
		ProvisioningPosture:      launcherbackend.ProvisioningPostureAttested,
		IsolateID:                isoID,
		SessionID:                sessionID,
		SessionNonce:             nonce,
		LaunchContextDigest:      syntheticLaunchContextDigest(spec, nonce),
		HandshakeTranscriptHash:  syntheticHandshakeTranscriptHash(spec, nonce, admission.DescriptorDigest),
		IsolateSessionKeyIDValue: syntheticSessionKeyIDValue(spec, nonce, admission.DescriptorDigest),
	}
	applyRuntimeAssetIdentity(&receipt, admission, qemuVersion, qemuBuild)
	applyLaunchExecutionDetails(&receipt, spec, cacheEvidence, qemuVersion, qemuBuild)
	if err := applyTrustedRuntimeAttestation(&receipt, admission, now); err != nil {
		return launcherbackend.BackendLaunchReceipt{}, err
	}
	return receipt, nil
}

func applyRuntimeAssetIdentity(receipt *launcherbackend.BackendLaunchReceipt, admission launcherbackend.RuntimeAdmissionRecord, qemuVersion, qemuBuild string) {
	receipt.HypervisorImplementation = launcherbackend.HypervisorImplementationQEMU
	receipt.AccelerationKind = launcherbackend.AccelerationKindKVM
	receipt.TransportKind = launcherbackend.TransportKindVirtioSerial
	receipt.QEMUProvenance = &launcherbackend.QEMUProvenance{Version: qemuVersion, BuildIdentity: qemuBuild}
	receipt.RuntimeImageDescriptorDigest = admission.DescriptorDigest
	receipt.RuntimeImageBootProfile = admission.BootContractVersion
	receipt.RuntimeImageSignerRef = admission.RuntimeImageSignerRef
	receipt.RuntimeImageVerifierRef = admission.RuntimeImageVerifierSetRef
	receipt.RuntimeImageSignatureDigest = admission.RuntimeImageSignatureDigest
	receipt.RuntimeToolchainDescriptorDigest = admission.RuntimeToolchainDescriptorDigest
	receipt.RuntimeToolchainSignerRef = admission.RuntimeToolchainSignerRef
	receipt.RuntimeToolchainVerifierRef = admission.RuntimeToolchainVerifierSetRef
	receipt.RuntimeToolchainSignatureDigest = admission.RuntimeToolchainSignatureDigest
	receipt.AuthorityStateDigest = admission.AuthorityStateDigest
	receipt.AuthorityStateRevision = admission.AuthorityStateRevision
	receipt.BootComponentDigestByName = cloneMap(admission.ComponentDigests)
	receipt.BootComponentDigests = componentDigestValues(admission.ComponentDigests)
}

func applyLaunchExecutionDetails(receipt *launcherbackend.BackendLaunchReceipt, spec launcherbackend.BackendLaunchSpec, cacheEvidence *launcherbackend.BackendCacheEvidence, qemuVersion, qemuBuild string) {
	receipt.QEMUProvenance = &launcherbackend.QEMUProvenance{Version: qemuVersion, BuildIdentity: qemuBuild}
	receipt.ResourceLimits = &spec.ResourceLimits
	receipt.WatchdogPolicy = &spec.WatchdogPolicy
	receipt.CachePosture = &spec.CachePosture
	receipt.CacheEvidence = cacheEvidence
	receipt.AttachmentPlanSummary = summarizeAttachments(spec.Attachments)
	receipt.WorkspaceEncryptionPosture = spec.Attachments.WorkspaceEncryption
	receipt.Lifecycle = &launcherbackend.BackendLifecycleSnapshot{CurrentState: launcherbackend.BackendLifecycleStateLaunching, TerminateBetweenSteps: true, TransitionCount: 1}
}

func syntheticLaunchContextDigest(spec launcherbackend.BackendLaunchSpec, nonce string) string {
	return syntheticDigest("launch-context", spec.RunID, spec.StageID, spec.RoleInstanceID, nonce)
}

func syntheticHandshakeTranscriptHash(spec launcherbackend.BackendLaunchSpec, nonce, descriptorDigest string) string {
	return syntheticDigest("handshake", spec.RunID, spec.StageID, spec.RoleInstanceID, nonce, descriptorDigest)
}

func syntheticSessionKeyIDValue(spec launcherbackend.BackendLaunchSpec, nonce, descriptorDigest string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{"session-key", spec.RunID, spec.StageID, spec.RoleInstanceID, nonce, descriptorDigest}, "|")))
	return hex.EncodeToString(sum[:])
}

func syntheticDigest(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func buildHardeningPosture() launcherbackend.AppliedHardeningPosture {
	return launcherbackend.AppliedHardeningPosture{
		Requested:                 launcherbackend.HardeningRequestedHardened,
		Effective:                 launcherbackend.HardeningEffectiveDegraded,
		DegradedReasons:           []string{"seccomp_unavailable"},
		ExecutionIdentityPosture:  launcherbackend.HardeningExecutionIdentityUnprivileged,
		FilesystemExposurePosture: launcherbackend.HardeningFilesystemExposureRestricted,
		NetworkExposurePosture:    launcherbackend.HardeningNetworkExposureNone,
		SyscallFilteringPosture:   launcherbackend.HardeningSyscallFilteringNone,
		DeviceSurfacePosture:      launcherbackend.HardeningDeviceSurfaceAllowlist,
		ControlChannelKind:        launcherbackend.TransportKindVirtioSerial,
		AccelerationKind:          launcherbackend.AccelerationKindKVM,
		BackendEvidenceRefs:       []string{"qemu:argv_allowlist_v1"},
	}
}

func (c *qemuController) prepareLaunchDir(spec launcherbackend.BackendLaunchSpec) (string, error) {
	root := strings.TrimSpace(c.cfg.WorkRoot)
	if root == "" {
		root = filepath.Join(os.TempDir(), "runecode-launcher")
	}
	cleanRoot := filepath.Clean(root)
	parentDir := filepath.Join(cleanRoot, safeToken(spec.RunID), safeToken(spec.StageID), safeToken(spec.RoleInstanceID))
	if rel, err := filepath.Rel(cleanRoot, parentDir); err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("launch path escaped work root")
	}
	if err := os.MkdirAll(parentDir, 0o700); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(parentDir, "launch-")
	if err != nil {
		return "", err
	}
	return dir, writeAttachmentManifests(dir, spec.Attachments)
}

func writeAttachmentManifests(launchDir string, plan launcherbackend.AttachmentPlan) error {
	attachmentsRoot := filepath.Join(launchDir, "attachments")
	if err := os.MkdirAll(attachmentsRoot, 0o700); err != nil {
		return err
	}
	for role, binding := range plan.ByRole {
		if err := writeAttachmentManifest(attachmentsRoot, role, binding); err != nil {
			return err
		}
	}
	return nil
}

func writeAttachmentManifest(root, role string, binding launcherbackend.AttachmentBinding) error {
	roleDir := filepath.Join(root, safeToken(role))
	if err := os.MkdirAll(roleDir, 0o700); err != nil {
		return err
	}
	payload := struct {
		Role            string   `json:"role"`
		ReadOnly        bool     `json:"read_only"`
		ChannelKind     string   `json:"channel_kind"`
		RequiredDigests []string `json:"required_digests,omitempty"`
	}{Role: role, ReadOnly: binding.ReadOnly, ChannelKind: binding.ChannelKind, RequiredDigests: append([]string{}, binding.RequiredDigests...)}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(roleDir, "manifest.json"), raw, 0o600)
}

func buildQEMUArgv(binary, kernel, initrd string, limits launcherbackend.BackendResourceLimits) []string {
	memory := limits.MemoryMiB
	if memory <= 0 {
		memory = 256
	}
	vcpus := limits.VCPUCount
	if vcpus <= 0 {
		vcpus = 1
	}
	return []string{
		binary,
		"-nodefaults",
		"-nographic",
		"-display", "none",
		"-serial", "stdio",
		"-no-reboot",
		"-machine", "q35,accel=kvm",
		"-cpu", "host",
		"-smp", fmt.Sprintf("%d", vcpus),
		"-m", fmt.Sprintf("%d", memory),
		"-nic", "none",
		"-kernel", kernel,
		"-initrd", initrd,
		"-append", "console=ttyS0 panic=-1",
	}
}

func summarizeAttachments(plan launcherbackend.AttachmentPlan) *launcherbackend.AttachmentPlanSummary {
	roles := make([]launcherbackend.AttachmentRoleSummary, 0, len(plan.ByRole))
	names := make([]string, 0, len(plan.ByRole))
	for role := range plan.ByRole {
		names = append(names, role)
	}
	sort.Strings(names)
	for _, role := range names {
		binding := plan.ByRole[role]
		roles = append(roles, launcherbackend.AttachmentRoleSummary{Role: role, ReadOnly: binding.ReadOnly, ChannelKind: binding.ChannelKind, DigestCount: len(binding.RequiredDigests)})
	}
	return &launcherbackend.AttachmentPlanSummary{Roles: roles, Constraints: plan.Constraints}
}

func detectQEMUProvenance(binary string) (string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, binary, "--version").Output()
	if err != nil {
		return "unknown", "qemu-system-x86_64"
	}
	line := strings.TrimSpace(string(bytes.SplitN(out, []byte("\n"), 2)[0]))
	version := "unknown"
	for _, part := range strings.Fields(line) {
		if strings.Contains(part, ".") && strings.Count(part, ".") >= 1 {
			version = strings.Trim(part, "()")
			break
		}
	}
	return version, line
}

func makeRuntimeIdentity(runID string) (string, string, string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", "", "", err
	}
	nonce := hex.EncodeToString(b)
	iso := "isolate-" + safeToken(runID) + "-" + nonce[:8]
	session := "session-" + nonce[8:16]
	return iso, session, nonce, nil
}

func cloneMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func safeToken(in string) string {
	v := strings.TrimSpace(strings.ToLower(in))
	if v == "" {
		return "x"
	}
	var b strings.Builder
	b.Grow(len(v))
	lastDash := false
	for i := 0; i < len(v); i++ {
		ch := v[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' {
			b.WriteByte(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	token := strings.Trim(b.String(), "-.")
	if token == "" {
		return "x"
	}
	return token
}
