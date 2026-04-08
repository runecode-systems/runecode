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
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
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

func putTrustedVerifierRecord(service *brokerapi.Service, record trustpolicy.VerifierRecord, importRequest trustedImportRequest) error {
	if _, err := trustpolicy.NewVerifierRegistry([]trustpolicy.VerifierRecord{record}); err != nil {
		return err
	}
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	predictedDigest, err := canonicalJSONDigestIdentity(b)
	if err != nil {
		return err
	}
	existing, err := trustedImportDigestExists(service, predictedDigest)
	if err != nil {
		return err
	}
	provenanceHash, err := trustedVerifierImportProvenanceDigest(record, importRequest)
	if err != nil {
		return err
	}
	ref, err := service.Put(artifacts.PutRequest{
		Payload:               b,
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
			artifacts.TrustedContractImportKindDetailKey:           artifacts.TrustedContractImportKindVerifierRecord,
			artifacts.TrustedContractImportArtifactDigestDetailKey: ref.Digest,
			artifacts.TrustedContractImportProvenanceDetailKey:     provenanceHash,
			"importer": importRequest.Importer.PrincipalID,
			"source":   importRequest.Source,
		},
	); err != nil {
		if !existing {
			cleanupErr := service.DeleteDigest(ref.Digest)
			if cleanupErr != nil {
				return fmt.Errorf("append trusted import audit event: %v (cleanup failed: %v)", err, cleanupErr)
			}
		}
		return err
	}
	return nil
}

func loadVerifierRecord(filePath string) (trustpolicy.VerifierRecord, error) {
	record := trustpolicy.VerifierRecord{}
	if strings.TrimSpace(filePath) == "" {
		return record, fmt.Errorf("path is required")
	}
	if err := loadJSONFileValue(filePath, &record); err != nil {
		return record, err
	}
	return record, nil
}

func loadTrustedImportRequest(filePath string) (trustedImportRequest, error) {
	request := trustedImportRequest{}
	if strings.TrimSpace(filePath) == "" {
		return request, fmt.Errorf("path is required")
	}
	if err := loadJSONFileValue(filePath, &request); err != nil {
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

func validateTrustedImportRequest(request trustedImportRequest) error {
	if request.SchemaID != "runecode.protocol.v0.TrustedContractImportRequest" {
		return fmt.Errorf("schema_id must be runecode.protocol.v0.TrustedContractImportRequest")
	}
	if request.SchemaVersion != "0.1.0" {
		return fmt.Errorf("schema_version must be 0.1.0")
	}
	if request.Kind != artifacts.TrustedContractImportKindVerifierRecord {
		return fmt.Errorf("kind must be %q", artifacts.TrustedContractImportKindVerifierRecord)
	}
	if err := validatePrincipalIdentity(request.Importer); err != nil {
		return fmt.Errorf("importer: %w", err)
	}
	if strings.TrimSpace(request.Reason) == "" {
		return fmt.Errorf("reason is required")
	}
	if strings.TrimSpace(request.ImportedAt) == "" {
		return fmt.Errorf("imported_at is required")
	}
	if strings.TrimSpace(request.Source) == "" {
		return fmt.Errorf("source is required")
	}
	return nil
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

func trustedVerifierImportProvenanceDigest(record trustpolicy.VerifierRecord, importRequest trustedImportRequest) (string, error) {
	payload := map[string]any{
		"schema_id":      "runecode.protocol.v0.TrustedContractImportReceipt",
		"schema_version": "0.1.0",
		"kind":           artifacts.TrustedContractImportKindVerifierRecord,
		"importer":       importRequest.Importer,
		"reason":         importRequest.Reason,
		"imported_at":    importRequest.ImportedAt,
		"source":         importRequest.Source,
		"verifier_record": map[string]string{
			"key_id":          record.KeyID,
			"key_id_value":    record.KeyIDValue,
			"logical_scope":   record.LogicalScope,
			"logical_purpose": record.LogicalPurpose,
		},
	}
	b, err := json.Marshal(payload)
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
