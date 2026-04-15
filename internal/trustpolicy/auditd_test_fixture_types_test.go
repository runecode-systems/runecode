package trustpolicy

import "crypto/ed25519"

type verifierStatusFixture struct {
	status          string
	statusChangedAt string
}

type auditVerificationFixture struct {
	segment              AuditSegmentFilePayload
	rawSegmentBytes      []byte
	sealEnvelope         SignedObjectEnvelope
	sealEnvelopeDigest   Digest
	verifierRecords      []VerifierRecord
	eventContractCatalog AuditEventContractCatalog
	signerEvidence       []AuditSignerEvidenceReference
	privateKey           ed25519.PrivateKey
	keyID                string
}

type verificationFixtureSignedArtifacts struct {
	segment            AuditSegmentFilePayload
	rawSegmentBytes    []byte
	sealEnvelope       SignedObjectEnvelope
	sealEnvelopeDigest Digest
}
