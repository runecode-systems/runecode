package brokerapi

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) trustedApprovalVerifiersForEnvelope(envelope trustpolicy.SignedObjectEnvelope) ([]trustpolicy.VerifierRecord, error) {
	records := make([]trustpolicy.VerifierRecord, 0)
	for _, artifactRecord := range s.List() {
		trusted, trustErr := s.isTrustedVerifierArtifact(artifactRecord)
		verifier, skip, err := selectVerifierRecordForEnvelope(artifactRecord, trusted, trustErr, envelope, s)
		if skip {
			if err != nil {
				return nil, err
			}
			continue
		}
		records = append(records, verifier)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("trusted verifier not found for signed approval decision")
	}
	return records, nil
}

func selectVerifierRecordForEnvelope(record artifacts.ArtifactRecord, trusted bool, trustErr error, envelope trustpolicy.SignedObjectEnvelope, service *Service) (trustpolicy.VerifierRecord, bool, error) {
	if trustErr != nil {
		return trustpolicy.VerifierRecord{}, true, trustErr
	}
	if !trusted {
		return trustpolicy.VerifierRecord{}, true, nil
	}
	verifier, err := service.loadVerifierRecord(record)
	if err != nil {
		return trustpolicy.VerifierRecord{}, true, err
	}
	if !matchesVerifierIdentity(verifier, envelope) {
		return trustpolicy.VerifierRecord{}, true, nil
	}
	if !isApprovalAuthorityVerifier(verifier) {
		return trustpolicy.VerifierRecord{}, true, nil
	}
	return verifier, false, nil
}

func (s *Service) loadVerifierRecord(record artifacts.ArtifactRecord) (trustpolicy.VerifierRecord, error) {
	reader, err := s.Get(record.Reference.Digest)
	if err != nil {
		return trustpolicy.VerifierRecord{}, fmt.Errorf("read trusted verifier artifact %q: %w", record.Reference.Digest, err)
	}
	blob, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		if readErr != nil {
			return trustpolicy.VerifierRecord{}, fmt.Errorf("read trusted verifier artifact %q bytes: %w", record.Reference.Digest, readErr)
		}
		return trustpolicy.VerifierRecord{}, fmt.Errorf("close trusted verifier artifact %q reader: %w", record.Reference.Digest, closeErr)
	}
	verifier := trustpolicy.VerifierRecord{}
	if err := json.Unmarshal(blob, &verifier); err != nil {
		return trustpolicy.VerifierRecord{}, fmt.Errorf("decode trusted verifier artifact %q: %w", record.Reference.Digest, err)
	}
	return verifier, nil
}

func matchesVerifierIdentity(verifier trustpolicy.VerifierRecord, envelope trustpolicy.SignedObjectEnvelope) bool {
	return verifier.KeyID == envelope.Signature.KeyID && verifier.KeyIDValue == envelope.Signature.KeyIDValue
}

func isApprovalAuthorityVerifier(verifier trustpolicy.VerifierRecord) bool {
	return verifier.LogicalPurpose == "approval_authority" && verifier.LogicalScope == "user"
}
