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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func buildLaunchReceipt(spec launcherbackend.BackendLaunchSpec, admission launcherbackend.RuntimeAdmissionRecord, isoID, sessionID, nonce, qemuVersion, qemuBuild string, cacheEvidence *launcherbackend.BackendCacheEvidence) launcherbackend.BackendLaunchReceipt {
	return launcherbackend.BackendLaunchReceipt{
		RunID:                            spec.RunID,
		StageID:                          spec.StageID,
		RoleInstanceID:                   spec.RoleInstanceID,
		RoleFamily:                       spec.RoleFamily,
		RoleKind:                         spec.RoleKind,
		BackendKind:                      launcherbackend.BackendKindMicroVM,
		IsolationAssuranceLevel:          launcherbackend.IsolationAssuranceIsolated,
		ProvisioningPosture:              launcherbackend.ProvisioningPostureTOFU,
		IsolateID:                        isoID,
		SessionID:                        sessionID,
		SessionNonce:                     nonce,
		HypervisorImplementation:         launcherbackend.HypervisorImplementationQEMU,
		AccelerationKind:                 launcherbackend.AccelerationKindKVM,
		TransportKind:                    launcherbackend.TransportKindVirtioSerial,
		QEMUProvenance:                   &launcherbackend.QEMUProvenance{Version: qemuVersion, BuildIdentity: qemuBuild},
		RuntimeImageDescriptorDigest:     admission.DescriptorDigest,
		RuntimeImageBootProfile:          admission.BootContractVersion,
		RuntimeImageSignerRef:            admission.RuntimeImageSignerRef,
		RuntimeImageVerifierRef:          admission.RuntimeImageVerifierSetRef,
		RuntimeImageSignatureDigest:      admission.RuntimeImageSignatureDigest,
		RuntimeToolchainDescriptorDigest: admission.RuntimeToolchainDescriptorDigest,
		RuntimeToolchainSignerRef:        admission.RuntimeToolchainSignerRef,
		RuntimeToolchainVerifierRef:      admission.RuntimeToolchainVerifierSetRef,
		RuntimeToolchainSignatureDigest:  admission.RuntimeToolchainSignatureDigest,
		BootComponentDigestByName:        cloneMap(admission.ComponentDigests),
		ResourceLimits:                   &spec.ResourceLimits,
		WatchdogPolicy:                   &spec.WatchdogPolicy,
		CachePosture:                     &spec.CachePosture,
		CacheEvidence:                    cacheEvidence,
		AttachmentPlanSummary:            summarizeAttachments(spec.Attachments),
		WorkspaceEncryptionPosture:       spec.Attachments.WorkspaceEncryption,
		Lifecycle:                        &launcherbackend.BackendLifecycleSnapshot{CurrentState: launcherbackend.BackendLifecycleStateLaunching, TerminateBetweenSteps: true, TransitionCount: 1},
	}
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
	dir := filepath.Join(cleanRoot, safeToken(spec.RunID), safeToken(spec.StageID), safeToken(spec.RoleInstanceID), fmt.Sprintf("launch-%d", c.cfg.Now().UnixNano()))
	if rel, err := filepath.Rel(cleanRoot, dir); err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("launch path escaped work root")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
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
func digestFileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
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
	v = strings.ReplaceAll(v, "/", "-")
	v = strings.ReplaceAll(v, "\\", "-")
	v = strings.ReplaceAll(v, " ", "-")
	return v
}

func backendError(code, msg string) error {
	if strings.TrimSpace(code) == "" {
		code = launcherbackend.BackendErrorCodeHypervisorLaunchFailed
	}
	if strings.TrimSpace(msg) == "" {
		msg = "backend launch failed"
	}
	return fmt.Errorf("backend_error_code=%s: %s", code, msg)
}
