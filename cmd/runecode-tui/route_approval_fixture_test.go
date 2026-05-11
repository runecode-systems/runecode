package main

import (
	"fmt"
	"strings"

	"github.com/runecode-systems/runecode/internal/trustpolicy"
)

var testSignedApprovalRequest = &trustpolicy.SignedObjectEnvelope{
	SchemaID:             trustpolicy.EnvelopeSchemaID,
	SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
	PayloadSchemaID:      trustpolicy.ApprovalRequestSchemaID,
	PayloadSchemaVersion: trustpolicy.ApprovalRequestSchemaVersion,
	Payload:              []byte(`{"schema_id":"runecode.protocol.v0.ApprovalRequest","schema_version":"0.3.0"}`),
	SignatureInput:       trustpolicy.SignatureInputProfile,
	Signature:            trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: strings.Repeat("a", 64), Signature: "c2ln"},
}

var testSignedApprovalDecision = testApprovalDecisionEnvelopeForRequest(testSignedApprovalRequest)

func testApprovalDecisionEnvelopeForRequest(request *trustpolicy.SignedObjectEnvelope) *trustpolicy.SignedObjectEnvelope {
	identity, err := approvalRequestDigestIdentity(*request)
	if err != nil {
		panic(err)
	}
	payload := fmt.Sprintf(`{"schema_id":"%s","schema_version":"%s","approval_request_hash":{"hash_alg":"sha256","hash":"%s"}}`, trustpolicy.ApprovalDecisionSchemaID, trustpolicy.ApprovalDecisionSchemaVersion, strings.TrimPrefix(identity, "sha256:"))
	return &trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      trustpolicy.ApprovalDecisionSchemaID,
		PayloadSchemaVersion: trustpolicy.ApprovalDecisionSchemaVersion,
		Payload:              []byte(payload),
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature:            trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: strings.Repeat("b", 64), Signature: "c2ln"},
	}
}
