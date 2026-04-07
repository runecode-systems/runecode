package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestAuditReadinessAndVerificationCommands(t *testing.T) {
	root := setBrokerServiceForTest(t)
	if err := seedLedgerForBrokerCommandTest(filepath.Join(root, "audit-ledger")); err != nil {
		t.Fatalf("seedLedgerForBrokerCommandTest returned error: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	if err := run([]string{"audit-readiness"}, stdout, stderr); err != nil {
		t.Fatalf("audit-readiness returned error: %v", err)
	}
	readiness := trustpolicy.AuditdReadiness{}
	if err := json.Unmarshal(stdout.Bytes(), &readiness); err != nil {
		t.Fatalf("audit-readiness output parse error: %v", err)
	}
	if !readiness.Ready {
		t.Fatal("readiness.ready = false, want true")
	}

	stdout.Reset()
	if err := run([]string{"audit-verification", "--limit", "5"}, stdout, stderr); err != nil {
		t.Fatalf("audit-verification returned error: %v", err)
	}
	surface := brokerapi.AuditVerificationSurface{}
	if err := json.Unmarshal(stdout.Bytes(), &surface); err != nil {
		t.Fatalf("audit-verification output parse error: %v", err)
	}
	if len(surface.Views) == 0 {
		t.Fatal("audit-verification views empty, want default operational view entries")
	}
}

func TestPromoteExcerptRequiresSignedApprovalInputs(t *testing.T) {
	setBrokerServiceForTest(t)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	unapprovedPath := writeTempFile(t, "excerpt.txt", "private excerpt")
	unapproved := putArtifactViaCLI(t, stdout, stderr, unapprovedPath, "unapproved_file_excerpts", testDigest("2"))
	err := run([]string{"promote-excerpt", "--unapproved-digest", unapproved.Digest, "--approver", "human", "--repo-path", "repo/file.txt", "--commit", "abc123", "--extractor-version", "tool-v1", "--full-content-visible"}, stdout, stderr)
	if err == nil {
		t.Fatal("promote-excerpt expected usage error when signed approval inputs are missing")
	}
	if _, ok := err.(*usageError); !ok {
		t.Fatalf("error type = %T, want *usageError", err)
	}
}

func TestPromoteExcerptRejectsSelfProvidedVerifierRecord(t *testing.T) {
	setBrokerServiceForTest(t)
	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	unapprovedPath := writeTempFile(t, "excerpt.txt", "private excerpt")
	unapproved := putArtifactViaCLI(t, stdout, stderr, unapprovedPath, "unapproved_file_excerpts", testDigest("2"))
	approvalRequestPath, approvalEnvelopePath, _ := writeApprovalFixtures(t, "human", unapproved.Digest, "repo/file.txt", "abc123", "tool-v1")
	_, _, verifierRecords := signedApprovalArtifactsForCLITests(t, "human", unapproved.Digest, "repo/file.txt", "abc123", "tool-v1")
	for index := range verifierRecords {
		payload, err := json.Marshal(verifierRecords[index])
		if err != nil {
			t.Fatalf("Marshal verifier error: %v", err)
		}
		payloadPath := writeTempFile(t, "verifier-non-auditd.json", string(payload))
		nibble := string('a' + rune(index%6))
		err = run([]string{"put-artifact", "--file", payloadPath, "--content-type", "application/json", "--data-class", "audit_verification_report", "--provenance-hash", testDigest(nibble), "--role", "workspace"}, stdout, stderr)
		if err != nil {
			t.Fatalf("put-artifact verifier record returned error: %v", err)
		}
	}
	err := run([]string{"promote-excerpt", "--unapproved-digest", unapproved.Digest, "--approver", "human", "--approval-request", approvalRequestPath, "--approval-envelope", approvalEnvelopePath, "--repo-path", "repo/file.txt", "--commit", "abc123", "--extractor-version", "tool-v1", "--full-content-visible"}, stdout, stderr)
	if err == nil {
		t.Fatal("promote-excerpt expected error when verifier records are not auditd-owned")
	}
}
