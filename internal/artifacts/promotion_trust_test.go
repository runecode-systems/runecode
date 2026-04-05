package artifacts

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestPromotionRejectsVerificationRecordFromNonAuditRole(t *testing.T) {
	store := newTestStore(t)
	unapproved, req := setupPromotionWithUntrustedVerifierForTests(t, store, "human-1")
	_, err := store.PromoteApprovedExcerpt(PromotionRequest{
		UnapprovedDigest:     unapproved.Digest,
		Approver:             "human-1",
		ApprovalDecision:     req.ApprovalDecision,
		RepoPath:             "repo/file.txt",
		Commit:               "abc123",
		ExtractorToolVersion: "v1",
		FullContentVisible:   true,
	})
	if !errors.Is(err, ErrVerifierNotFound) {
		t.Fatalf("Promote error = %v, want ErrVerifierNotFound", err)
	}
}

func seedTrustedVerifierRecordsForTests(t *testing.T, store *Store, verifiers []trustpolicy.VerifierRecord) {
	t.Helper()
	for index := range verifiers {
		payload, err := json.Marshal(verifiers[index])
		if err != nil {
			t.Fatalf("Marshal verifier record error: %v", err)
		}
		nibble := string('a' + rune(index%6))
		if _, err := store.Put(PutRequest{
			Payload:               payload,
			ContentType:           "application/json",
			DataClass:             DataClassAuditVerificationReport,
			ProvenanceReceiptHash: testDigest(nibble),
			CreatedByRole:         "auditd",
			TrustedSource:         true,
		}); err != nil {
			t.Fatalf("Put verifier record error: %v", err)
		}
	}
}

func setupPromotionSourceForTests(t *testing.T) (*Store, ArtifactReference) {
	t.Helper()
	store := newTestStore(t)
	unapproved, err := store.Put(PutRequest{
		Payload:               []byte("sensitive excerpt"),
		ContentType:           "text/plain",
		DataClass:             DataClassUnapprovedFileExcerpts,
		ProvenanceReceiptHash: testDigest("5"),
		CreatedByRole:         "workspace",
	})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	return store, unapproved
}

func assertPromotionRequiresApprover(t *testing.T, store *Store, digest string) {
	t.Helper()
	_, err := store.PromoteApprovedExcerpt(PromotionRequest{UnapprovedDigest: digest})
	if err != ErrPromotionRequiresApproval {
		t.Fatalf("Promote no approver error = %v, want %v", err, ErrPromotionRequiresApproval)
	}
}

func promoteApprovedExcerptForTests(t *testing.T, store *Store, digest string, approver string) ArtifactReference {
	t.Helper()
	ref, err := promoteApprovedExcerptWithFlagsForTests(t, store, digest, approver, false, false)
	if err != nil {
		t.Fatalf("Promote returned error: %v", err)
	}
	return ref
}

func promoteApprovedExcerptWithFlagsForTests(t *testing.T, store *Store, digest string, approver string, bulk bool, bulkApproved bool) (ArtifactReference, error) {
	t.Helper()
	req, verifiers, err := signedPromotionRequestForTests(approver)
	if err != nil {
		t.Fatalf("signedPromotionRequestForTests error: %v", err)
	}
	seedTrustedVerifierRecordsForTests(t, store, verifiers)
	return store.PromoteApprovedExcerpt(PromotionRequest{
		UnapprovedDigest:      digest,
		Approver:              approver,
		ApprovalDecision:      req.ApprovalDecision,
		RepoPath:              "a",
		Commit:                "b",
		ExtractorToolVersion:  "c",
		FullContentVisible:    true,
		BulkRequest:           bulk,
		BulkApprovalConfirmed: bulkApproved,
	})
}

func setupPromotionWithUntrustedVerifierForTests(t *testing.T, store *Store, approver string) (ArtifactReference, PromotionRequest) {
	t.Helper()
	req, verifiers, err := signedPromotionRequestForTests(approver)
	if err != nil {
		t.Fatalf("signedPromotionRequestForTests error: %v", err)
	}
	payload, err := json.Marshal(verifiers[0])
	if err != nil {
		t.Fatalf("Marshal verifier record error: %v", err)
	}
	if _, err := store.Put(PutRequest{
		Payload:               payload,
		ContentType:           "application/json",
		DataClass:             DataClassAuditVerificationReport,
		ProvenanceReceiptHash: testDigest("a"),
		CreatedByRole:         "workspace",
	}); err != nil {
		t.Fatalf("Put verifier record error: %v", err)
	}
	unapproved, err := store.Put(PutRequest{
		Payload:               []byte("sensitive excerpt"),
		ContentType:           "text/plain",
		DataClass:             DataClassUnapprovedFileExcerpts,
		ProvenanceReceiptHash: testDigest("b"),
		CreatedByRole:         "workspace",
	})
	if err != nil {
		t.Fatalf("Put unapproved artifact error: %v", err)
	}
	return unapproved, req
}
