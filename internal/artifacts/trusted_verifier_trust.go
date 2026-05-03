package artifacts

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func IsTrustedVerifierArtifact(record ArtifactRecord, events []AuditEvent) bool {
	if record.Reference.DataClass != DataClassAuditVerificationReport {
		return false
	}
	if !isValidTrustedVerifierProvenance(record.Reference.ProvenanceReceiptHash) {
		return false
	}
	switch record.CreatedByRole {
	case "auditd":
		return hasAuditEventArtifactPutForRole(events, record.Reference.Digest, record.CreatedByRole, record.Reference.ProvenanceReceiptHash)
	case "broker":
		return hasTrustedVerifierImportAuditEvent(events, record.Reference.Digest, record.Reference.ProvenanceReceiptHash)
	default:
		return false
	}
}

func isValidTrustedVerifierProvenance(value string) bool {
	_, err := digestFromIdentityString(value)
	return err == nil
}

func digestFromIdentityString(value string) (trustpolicy.Digest, error) {
	parts := splitDigestIdentity(value)
	if len(parts) != 2 {
		return trustpolicy.Digest{}, fmt.Errorf("digest identity must be hash_alg:hash")
	}
	digest := trustpolicy.Digest{HashAlg: parts[0], Hash: parts[1]}
	_, err := digest.Identity()
	return digest, err
}

func splitDigestIdentity(value string) []string {
	return strings.SplitN(strings.TrimSpace(value), ":", 2)
}

func hasAuditEventArtifactPutForRole(events []AuditEvent, digest, role, provenanceHash string) bool {
	for _, event := range events {
		if event.Type != "artifact_put" || event.Actor != role {
			continue
		}
		if !auditEventDetailMatches(event.Details, "digest", digest) {
			continue
		}
		if !auditEventDetailMatches(event.Details, "data_class", string(DataClassAuditVerificationReport)) {
			continue
		}
		if !auditEventDetailMatches(event.Details, "provenance_receipt_hash", provenanceHash) {
			continue
		}
		return true
	}
	return false
}

func hasTrustedVerifierImportAuditEvent(events []AuditEvent, artifactDigest, provenanceHash string) bool {
	for _, event := range events {
		if event.Type != TrustedContractImportAuditEventType || event.Actor != "brokerapi" {
			continue
		}
		if !auditEventDetailMatches(event.Details, TrustedContractImportKindDetailKey, TrustedContractImportKindVerifierRecord) {
			continue
		}
		if !auditEventDetailMatches(event.Details, TrustedContractImportArtifactDigestDetailKey, artifactDigest) {
			continue
		}
		if !auditEventDetailMatches(event.Details, TrustedContractImportProvenanceDetailKey, provenanceHash) {
			continue
		}
		return true
	}
	return false
}

func auditEventDetailMatches(details map[string]interface{}, key, want string) bool {
	value, ok := details[key].(string)
	if !ok {
		return false
	}
	return value == want
}
