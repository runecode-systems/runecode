package brokerapi

import (
	"encoding/json"
	"fmt"

	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func (s *Service) signAuditEvidenceBundleManifestEnvelope(manifest AuditEvidenceBundleManifest) (trustpolicy.SignedObjectEnvelope, error) {
	if s == nil || s.secretsSvc == nil {
		return trustpolicy.SignedObjectEnvelope{}, fmt.Errorf("secrets service unavailable")
	}
	payloadBytes, canonical, err := auditEvidenceBundleManifestEnvelopeBytes(manifest)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	if s.auditLedger == nil {
		return trustpolicy.SignedObjectEnvelope{}, fmt.Errorf("audit ledger unavailable")
	}
	_, targetSealDigest, err := s.auditLedger.LatestAnchorableSeal()
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	presence, err := s.externalAnchorPresenceAttestation(targetSealDigest)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	signed, err := s.secretsSvc.SignAuditAnchor(secretsd.AuditAnchorSignRequest{
		PayloadCanonicalBytes: canonical,
		TargetSealDigest:      targetSealDigest,
		LogicalScope:          "node",
		PresenceAttestation:   presence,
	})
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	return trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      "runecode.protocol.v0.AuditEvidenceBundleManifest",
		PayloadSchemaVersion: "0.1.0",
		Payload:              payloadBytes,
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature:            signed.Signature,
	}, nil
}

func auditEvidenceBundleManifestEnvelopeBytes(manifest AuditEvidenceBundleManifest) ([]byte, []byte, error) {
	b, err := json.Marshal(manifest)
	if err != nil {
		return nil, nil, err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return nil, nil, err
	}
	return b, canonical, nil
}
