package brokerapi

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func buildGitCredentialEnv(workdir, token string) ([]string, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("token is required")
	}
	secretsDir := filepath.Join(workdir, ".runecode")
	if err := os.MkdirAll(secretsDir, 0o700); err != nil {
		return nil, err
	}
	tokenFile := filepath.Join(secretsDir, "token")
	if err := os.WriteFile(tokenFile, []byte(token), 0o600); err != nil {
		return nil, err
	}
	askPass := filepath.Join(secretsDir, "askpass.sh")
	askPassScript := "#!/bin/sh\ncase \"$1\" in\n  *Username*) printf '%s\\n' 'x-access-token' ;;\n  *) cat \"$RUNE_GIT_TOKEN_FILE\" ;;\nesac\n"
	if err := os.WriteFile(askPass, []byte(askPassScript), 0o700); err != nil {
		return nil, err
	}
	return []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=" + askPass,
		"RUNE_GIT_TOKEN_FILE=" + tokenFile,
	}, nil
}

func runGit(ctx context.Context, workdir string, env []string, stdin io.Reader, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workdir
	cmd.Env = gitCommandEnv(env)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("git %s failed: %s", strings.Join(args, " "), msg)
	}
	return nil
}

func runGitOutput(ctx context.Context, workdir string, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workdir
	cmd.Env = gitCommandEnv(env)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func lsRemoteOID(ctx context.Context, workdir string, env []string, remoteURL, ref string) (string, error) {
	out, err := runGitOutput(ctx, workdir, env, "ls-remote", remoteURL, ref)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(out) == "" {
		return "", nil
	}
	scanner := bufio.NewScanner(strings.NewReader(out))
	if !scanner.Scan() {
		return "", nil
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) == 0 {
		return "", nil
	}
	return strings.TrimSpace(fields[0]), nil
}

func applyPatchArtifacts(ctx context.Context, service *Service, workdir string, env []string, digests []trustpolicy.Digest) *gitRemoteExecutionError {
	if len(digests) == 0 {
		return executionFailure("broker_validation_schema_invalid", "validation", "git_patch_artifacts_missing", "referenced patch artifacts are required")
	}
	for _, digest := range digests {
		patchBytes, errResp := readPatchArtifact(service, digest)
		if errResp != nil {
			return errResp
		}
		if err := runGit(ctx, workdir, env, bytes.NewReader(patchBytes), "apply", "--index", "--allow-empty", "--whitespace=nowarn", "-"); err != nil {
			return executionFailure("gateway_failure", "internal", "git_patch_apply_failed", err.Error())
		}
	}
	return nil
}

func readPatchArtifact(service *Service, digest trustpolicy.Digest) ([]byte, *gitRemoteExecutionError) {
	identity, err := digest.Identity()
	if err != nil {
		return nil, executionFailure("broker_validation_schema_invalid", "validation", "git_patch_artifact_digest_invalid", "patch artifact digest invalid")
	}
	record, err := service.Head(identity)
	if err != nil {
		return nil, executionFailure("broker_not_found_prepared_mutation", "storage", "git_patch_artifact_missing", fmt.Sprintf("patch artifact %s not found", identity))
	}
	if record.Reference.DataClass != artifacts.DataClassDiffs {
		return nil, executionFailure("broker_validation_schema_invalid", "validation", "git_patch_artifact_data_class_invalid", "patch artifacts must be in diffs data class")
	}
	reader, err := service.Get(identity)
	if err != nil {
		return nil, executionFailure("broker_not_found_prepared_mutation", "storage", "git_patch_artifact_missing", fmt.Sprintf("patch artifact %s unavailable", identity))
	}
	patchBytes, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if readErr != nil {
		return nil, executionFailure("gateway_failure", "internal", "git_patch_artifact_read_failed", readErr.Error())
	}
	return patchBytes, nil
}

func commitWithIntent(ctx context.Context, workdir string, env []string, intent gitCommitIntent) error {
	message := commitMessageFromIntent(intent)
	commitEnv := append([]string{}, env...)
	commitEnv = append(commitEnv, commitIdentityEnv(intent)...)
	return runGit(ctx, workdir, commitEnv, strings.NewReader(message), "commit", "--file", "-")
}

func commitMessageFromIntent(intent gitCommitIntent) string {
	subject := firstNonEmpty(strings.TrimSpace(intent.Message.Subject), "Apply approved patch")
	body := strings.TrimSpace(intent.Message.Body)
	message := subject
	if body != "" {
		message += "\n\n" + body
	}
	trailers := commitTrailers(intent.Trailers)
	if len(trailers) > 0 {
		message += "\n\n" + strings.Join(trailers, "\n")
	}
	return message
}

func commitTrailers(trailers []struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}) []string {
	lines := make([]string, 0, len(trailers))
	for _, trailer := range trailers {
		key := strings.TrimSpace(trailer.Key)
		val := strings.TrimSpace(trailer.Value)
		if key == "" || val == "" {
			continue
		}
		lines = append(lines, key+": "+val)
	}
	return lines
}

func commitIdentityEnv(intent gitCommitIntent) []string {
	name := firstNonEmpty(strings.TrimSpace(intent.Author.DisplayName), "RuneCode Gateway")
	email := firstNonEmpty(strings.TrimSpace(intent.Author.Email), "gateway@runecode.invalid")
	return []string{
		"GIT_AUTHOR_NAME=" + name,
		"GIT_AUTHOR_EMAIL=" + email,
		"GIT_COMMITTER_NAME=" + firstNonEmpty(strings.TrimSpace(intent.Committer.DisplayName), name),
		"GIT_COMMITTER_EMAIL=" + firstNonEmpty(strings.TrimSpace(intent.Committer.Email), email),
	}
}

func commitSimple(ctx context.Context, workdir string, env []string, title, body string) error {
	intent := gitCommitIntent{}
	intent.Message.Subject = firstNonEmpty(strings.TrimSpace(title), "Apply approved pull-request patch")
	intent.Message.Body = strings.TrimSpace(body)
	return commitWithIntent(ctx, workdir, env, intent)
}

func verifyExpectedResultTree(ctx context.Context, workdir string, env []string, expected trustpolicy.Digest) (trustpolicy.Digest, *gitRemoteExecutionError) {
	expectedIdentity, err := expected.Identity()
	if err != nil {
		return trustpolicy.Digest{}, executionFailure("broker_validation_schema_invalid", "validation", "expected_result_tree_hash_invalid", "expected_result_tree_hash invalid")
	}
	treeOID, err := runGitOutput(ctx, workdir, env, "rev-parse", "HEAD^{tree}")
	if err != nil {
		return trustpolicy.Digest{}, executionFailure("gateway_failure", "internal", "git_tree_hash_unavailable", err.Error())
	}
	h := sha256.Sum256([]byte(strings.TrimSpace(treeOID)))
	observed := trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(h[:])}
	observedIdentity, _ := observed.Identity()
	if observedIdentity != expectedIdentity {
		return observed, executionFailureWithState("broker_approval_state_invalid", "auth", "git_result_tree_hash_mismatch", "observed result tree hash does not match expected_result_tree_hash", gitRemoteMutationExecutionBlocked)
	}
	return observed, nil
}

func comparableObjectIdentity(rawOID, expected string) string {
	oid := strings.TrimSpace(rawOID)
	if oid == "" {
		return gitRemoteMutationZeroObjectID
	}
	if len(strings.TrimSpace(expected)) == len(oid) {
		return oid
	}
	h := sha256.Sum256([]byte(oid))
	return hex.EncodeToString(h[:])
}

func executionFailure(code, category, reason, message string) *gitRemoteExecutionError {
	return executionFailureWithState(code, category, reason, message, gitRemoteMutationExecutionFailed)
}

func executionFailureWithState(code, category, reason, message, state string) *gitRemoteExecutionError {
	return &gitRemoteExecutionError{
		code:           code,
		category:       category,
		reasonCode:     reason,
		message:        message,
		retryable:      false,
		executionState: state,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func gitCommandEnv(extra []string) []string {
	base := []string{"PATH=" + firstNonEmpty(os.Getenv("PATH"), "/usr/bin:/bin")}
	if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
		base = append(base, "HOME="+home)
	}
	base = append(base,
		"LANG=C",
		"LC_ALL=C",
		"GIT_TERMINAL_PROMPT=0",
	)
	return append(base, extra...)
}
