package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func defaultBrokerStoreRoot() string {
	cacheDir, err := os.UserCacheDir()
	if err == nil && cacheDir != "" {
		return filepath.Join(cacheDir, "runecode", "artifact-store")
	}
	configDir, configErr := os.UserConfigDir()
	if configErr == nil && configDir != "" {
		return filepath.Join(configDir, "runecode", "artifact-store")
	}
	return filepath.Join(os.TempDir(), "runecode", "artifact-store")
}

func writeJSON(w io.Writer, value interface{}) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

type trustedImportRequest struct {
	SchemaID      string                        `json:"schema_id"`
	SchemaVersion string                        `json:"schema_version"`
	Kind          string                        `json:"kind"`
	Importer      trustpolicy.PrincipalIdentity `json:"importer"`
	Reason        string                        `json:"reason"`
	ImportedAt    string                        `json:"imported_at"`
	Source        string                        `json:"source"`
}

func putTrustedContractArtifact(service *brokerapi.Service, kind string, payload []byte, importRequest trustedImportRequest) error {
	if err := validateTrustedContractPayload(kind, payload); err != nil {
		return err
	}
	predictedDigest, err := canonicalJSONDigestIdentity(payload)
	if err != nil {
		return err
	}
	existing, err := trustedImportDigestExists(service, predictedDigest)
	if err != nil {
		return err
	}
	provenanceHash, err := trustedContractImportProvenanceDigest(kind, payload, importRequest)
	if err != nil {
		return err
	}
	ref, err := service.Put(artifacts.PutRequest{
		Payload:               payload,
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassAuditVerificationReport,
		ProvenanceReceiptHash: provenanceHash,
		CreatedByRole:         "broker",
		TrustedSource:         true,
	})
	if err != nil {
		return err
	}
	if err := service.AppendTrustedAuditEvent(
		artifacts.TrustedContractImportAuditEventType,
		"brokerapi",
		map[string]interface{}{
			artifacts.TrustedContractImportKindDetailKey:           kind,
			artifacts.TrustedContractImportArtifactDigestDetailKey: ref.Digest,
			artifacts.TrustedContractImportProvenanceDetailKey:     provenanceHash,
			"importer": importRequest.Importer.PrincipalID,
			"source":   importRequest.Source,
		},
	); err != nil {
		if !existing {
			return fmt.Errorf("append trusted import audit event: %w (artifact persisted; retry import to finalize trust admission)", err)
		}
		return err
	}
	return nil
}

func loadTrustedContractPayload(filePath string) ([]byte, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("path is required")
	}
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(b))
	if trimmed == "" {
		return nil, fmt.Errorf("payload is required")
	}
	return []byte(trimmed), nil
}

func loadTrustedImportRequest(filePath string) (trustedImportRequest, error) {
	request := trustedImportRequest{}
	if strings.TrimSpace(filePath) == "" {
		return request, fmt.Errorf("path is required")
	}
	if err := loadStrictJSONFileValue(filePath, &request); err != nil {
		return request, err
	}
	if err := validateTrustedImportRequest(request); err != nil {
		return request, err
	}
	return request, nil
}

func loadJSONFileValue(filePath string, target any) error {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

func loadStrictJSONFileValue(filePath string, target any) error {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(b)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON content")
		}
		return err
	}
	return nil
}

func validateTrustedImportRequest(request trustedImportRequest) error {
	if request.SchemaID != "runecode.protocol.v0.TrustedContractImportRequest" {
		return fmt.Errorf("schema_id must be runecode.protocol.v0.TrustedContractImportRequest")
	}
	if request.SchemaVersion != "0.1.0" {
		return fmt.Errorf("schema_version must be 0.1.0")
	}
	if err := validateTrustedImportKind(request.Kind); err != nil {
		return err
	}
	if err := validatePrincipalIdentity(request.Importer); err != nil {
		return fmt.Errorf("importer: %w", err)
	}
	if strings.TrimSpace(request.Reason) == "" {
		return fmt.Errorf("reason is required")
	}
	if len(request.Reason) > 512 {
		return fmt.Errorf("reason must be <= 512 characters")
	}
	if strings.TrimSpace(request.ImportedAt) == "" {
		return fmt.Errorf("imported_at is required")
	}
	parsedImportedAt, err := time.Parse(time.RFC3339, request.ImportedAt)
	if err != nil {
		return fmt.Errorf("imported_at must be RFC3339: %w", err)
	}
	if parsedImportedAt.Format(time.RFC3339) != request.ImportedAt {
		return fmt.Errorf("imported_at must use canonical RFC3339 form")
	}
	if strings.TrimSpace(request.Source) == "" {
		return fmt.Errorf("source is required")
	}
	if len(request.Source) > 256 {
		return fmt.Errorf("source must be <= 256 characters")
	}
	return nil
}

func validateTrustedImportKind(kind string) error {
	if _, ok := trustedContractSchemaPathByKind[strings.TrimSpace(kind)]; ok {
		return nil
	}
	return fmt.Errorf("kind must be one of: %s", strings.Join(supportedTrustedImportKinds(), ", "))
}

func validatePrincipalIdentity(identity trustpolicy.PrincipalIdentity) error {
	if identity.SchemaID != "runecode.protocol.v0.PrincipalIdentity" {
		return fmt.Errorf("schema_id must be runecode.protocol.v0.PrincipalIdentity")
	}
	if identity.SchemaVersion != "0.2.0" {
		return fmt.Errorf("schema_version must be 0.2.0")
	}
	if strings.TrimSpace(identity.ActorKind) == "" {
		return fmt.Errorf("actor_kind is required")
	}
	if strings.TrimSpace(identity.PrincipalID) == "" {
		return fmt.Errorf("principal_id is required")
	}
	if strings.TrimSpace(identity.InstanceID) == "" {
		return fmt.Errorf("instance_id is required")
	}
	return nil
}

func trustedContractImportProvenanceDigest(kind string, payload []byte, importRequest trustedImportRequest) (string, error) {
	payloadDigest, err := canonicalJSONDigestIdentity(payload)
	if err != nil {
		return "", err
	}
	receiptPayload := map[string]any{
		"schema_id":      "runecode.protocol.v0.TrustedContractImportReceipt",
		"schema_version": "0.1.0",
		"kind":           kind,
		"importer":       importRequest.Importer,
		"reason":         importRequest.Reason,
		"imported_at":    importRequest.ImportedAt,
		"source":         importRequest.Source,
		"payload_digest": payloadDigest,
	}
	b, err := json.Marshal(receiptPayload)
	if err != nil {
		return "", err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

var trustedContractSchemaPathByKind = map[string]string{
	artifacts.TrustedContractImportKindVerifierRecord:  "objects/VerifierRecord.schema.json",
	artifacts.TrustedContractImportKindRoleManifest:    "objects/RoleManifest.schema.json",
	artifacts.TrustedContractImportKindRunCapability:   "objects/CapabilityManifest.schema.json",
	artifacts.TrustedContractImportKindStageCapability: "objects/CapabilityManifest.schema.json",
	artifacts.TrustedContractImportKindPolicyAllowlist: "objects/PolicyAllowlist.schema.json",
	artifacts.TrustedContractImportKindPolicyRuleSet:   "objects/PolicyRuleSet.schema.json",
}

func supportedTrustedImportKinds() []string {
	kinds := make([]string, 0, len(trustedContractSchemaPathByKind))
	for kind := range trustedContractSchemaPathByKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

func validateTrustedContractPayload(kind string, payload []byte) error {
	trimmedKind := strings.TrimSpace(kind)
	schemaPath, ok := trustedContractSchemaPathByKind[trimmedKind]
	if !ok {
		return validateTrustedImportKind(trimmedKind)
	}
	if err := policyengine.ValidateObjectPayloadAgainstSchema(payload, schemaPath); err != nil {
		return err
	}
	if trimmedKind == artifacts.TrustedContractImportKindVerifierRecord {
		record := trustpolicy.VerifierRecord{}
		if err := json.Unmarshal(payload, &record); err != nil {
			return err
		}
		if _, err := trustpolicy.NewVerifierRegistry([]trustpolicy.VerifierRecord{record}); err != nil {
			return err
		}
	}
	if trimmedKind == artifacts.TrustedContractImportKindRunCapability || trimmedKind == artifacts.TrustedContractImportKindStageCapability {
		manifest := policyengine.CapabilityManifest{}
		if err := json.Unmarshal(payload, &manifest); err != nil {
			return err
		}
		wantScope := "run"
		if trimmedKind == artifacts.TrustedContractImportKindStageCapability {
			wantScope = "stage"
		}
		if strings.TrimSpace(manifest.ManifestScope) != wantScope {
			return fmt.Errorf("manifest_scope must be %q for %s", wantScope, trimmedKind)
		}
	}
	return nil
}

func canonicalJSONDigestIdentity(payload []byte) (string, error) {
	canonical, err := jsoncanonicalizer.Transform(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func trustedImportDigestExists(service *brokerapi.Service, digest string) (bool, error) {
	_, err := service.Head(digest)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, artifacts.ErrArtifactNotFound) {
		return false, nil
	}
	return false, err
}

func loadSignedApprovalEnvelope(filePath string) (*trustpolicy.SignedObjectEnvelope, error) {
	if filePath == "" {
		return nil, fmt.Errorf("path is required")
	}
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	envelope := trustpolicy.SignedObjectEnvelope{}
	if err := json.Unmarshal(b, &envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}

func defaultRequestID() string {
	return fmt.Sprintf("cli-%d", time.Now().UTC().UnixNano())
}
