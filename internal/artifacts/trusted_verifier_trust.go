package artifacts

import (
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
	digest := trustpolicy.Digest{HashAlg: "sha256", Hash: trimDigestPrefix(value)}
	_, err := digest.Identity()
	return err == nil
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

func trimDigestPrefix(value string) string {
	const prefix = "sha256:"
	if len(value) > len(prefix) && value[:len(prefix)] == prefix {
		return value[len(prefix):]
	}
	return value
}
