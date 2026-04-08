package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
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

func putTrustedVerifierRecord(service *brokerapi.Service, record trustpolicy.VerifierRecord) error {
	if _, err := trustpolicy.NewVerifierRegistry([]trustpolicy.VerifierRecord{record}); err != nil {
		return err
	}
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = service.Put(artifacts.PutRequest{
		Payload:               b,
		ContentType:           "application/json",
		DataClass:             artifacts.DataClassAuditVerificationReport,
		ProvenanceReceiptHash: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		CreatedByRole:         "auditd",
		TrustedSource:         true,
	})
	return err
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

func loadJSONFileValue(filePath string, target any) error {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
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
