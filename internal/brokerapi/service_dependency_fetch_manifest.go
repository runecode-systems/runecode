package brokerapi

import (
	"encoding/json"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func (s *dependencyFetchService) resolvedManifestObject(req DependencyFetchRequestObject, requestHash string, unit artifacts.DependencyCacheResolvedUnitRecord) (map[string]any, error) {
	if len(unit.PayloadDigest) == 0 {
		return nil, artifacts.ErrDependencyCacheIncompleteState
	}
	payloadArtifact, err := s.owner.Head(unit.PayloadDigest[0])
	if err != nil {
		return nil, err
	}
	return s.buildResolvedUnitManifestPayload(req, requestHash, unit.ResolvedUnitDigest, payloadArtifact.Reference, dependencyRegistryFetchMetadata{})
}

func (s *dependencyFetchService) buildResolvedUnitManifestPayload(req DependencyFetchRequestObject, requestHash, resolvedDigest string, payloadRef artifacts.ArtifactReference, metadata dependencyRegistryFetchMetadata) (map[string]any, error) {
	payload := map[string]any{
		"schema_id":            "runecode.protocol.v0.DependencyResolvedUnitManifest",
		"schema_version":       "0.1.0",
		"request_hash":         digestObjectForIdentity(requestHash),
		"resolved_unit_digest": digestObjectForIdentity(resolvedDigest),
		"dependency_request":   req,
		"payload_artifacts": []any{map[string]any{
			"schema_id":               "runecode.protocol.v0.ArtifactReference",
			"schema_version":          "0.4.0",
			"digest":                  digestObjectForIdentity(payloadRef.Digest),
			"size_bytes":              payloadRef.SizeBytes,
			"content_type":            payloadRef.ContentType,
			"data_class":              string(payloadRef.DataClass),
			"provenance_receipt_hash": digestObjectForIdentity(payloadRef.ProvenanceReceiptHash),
		}},
		"integrity": map[string]any{
			"verification_state": "verified",
		},
		"materialization": map[string]any{
			"derived_only":       true,
			"read_only_required": true,
		},
	}
	if digest := strings.TrimSpace(metadata.UpstreamManifestDigest); digest != "" {
		payload["integrity"].(map[string]any)["upstream_manifest_digest"] = digestObjectForIdentity(digest)
	}
	if err := validateJSONEnvelope(payload, dependencyResolvedUnitManifestSchemaPath); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *dependencyFetchService) computeResolvedUnitDigest(req DependencyFetchRequestObject, requestHash, payloadDigest string) (string, error) {
	input := map[string]any{
		"request_hash":       requestHash,
		"dependency_request": req,
		"payload_digests":    []string{payloadDigest},
	}
	return canonicalDigestIdentity(input)
}

func canonicalDigestIdentity(value any) (string, error) {
	d, err := canonicalDigestForValue(value)
	if err != nil {
		return "", err
	}
	return d.Identity()
}

func canonicalDependencyRequestIdentity(req DependencyFetchRequestObject) (string, error) {
	identity := map[string]any{
		"schema_id":         req.SchemaID,
		"schema_version":    req.SchemaVersion,
		"request_kind":      req.RequestKind,
		"registry_identity": req.RegistryIdentity,
		"ecosystem":         req.Ecosystem,
		"package_name":      req.PackageName,
		"package_version":   req.PackageVersion,
	}
	return canonicalDigestIdentity(identity)
}

func canonicalDependencyBatchIdentity(req DependencyFetchBatchRequestObject) (string, error) {
	identity := map[string]any{
		"schema_id":           req.SchemaID,
		"schema_version":      req.SchemaVersion,
		"lockfile_kind":       req.LockfileKind,
		"lockfile_digest":     req.LockfileDigest,
		"request_set_hash":    req.RequestSetHash,
		"dependency_requests": req.DependencyRequests,
	}
	return canonicalDigestIdentity(identity)
}

func coalesceString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func canonicalJSONBytesForValue(value any) ([]byte, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return jsoncanonicalizer.Transform(b)
}

func mapDigestIdentities(identities []string) []trustpolicy.Digest {
	out := make([]trustpolicy.Digest, 0, len(identities))
	for _, identity := range identities {
		out = append(out, mustDigestObjectFromIdentity(identity))
	}
	return out
}

func mustDigestObjectFromIdentity(identity string) trustpolicy.Digest {
	d, err := digestFromIdentity(identity)
	if err != nil {
		panic(err)
	}
	return d
}

func mustDigestIdentity(d trustpolicy.Digest) string {
	identity, err := d.Identity()
	if err != nil {
		panic(err)
	}
	return identity
}
