package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func handleAuditEvidenceSnapshotGet(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("audit-evidence-snapshot-get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "audit-evidence-snapshot-get usage: runecode-broker audit-evidence-snapshot-get"}
	}
	if len(fs.Args()) != 0 {
		return &usageError{message: "audit-evidence-snapshot-get does not accept positional arguments"}
	}

	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.AuditEvidenceSnapshotGet(ctx, brokerapi.AuditEvidenceSnapshotGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceSnapshotGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp.Snapshot)
}

func handleAuditEvidenceRetentionReview(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("audit-evidence-retention-review", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	requestFile := fs.String("request-file", "", "path to retention review request JSON")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "audit-evidence-retention-review usage: runecode-broker audit-evidence-retention-review --request-file path"}
	}
	if strings.TrimSpace(*requestFile) == "" {
		return &usageError{message: "audit-evidence-retention-review requires --request-file"}
	}
	if len(fs.Args()) != 0 {
		return &usageError{message: "audit-evidence-retention-review does not accept positional arguments"}
	}
	req := brokerapi.AuditEvidenceRetentionReviewRequest{}
	if err := loadStrictJSONFileValue(strings.TrimSpace(*requestFile), &req); err != nil {
		return err
	}
	req.SchemaID = "runecode.protocol.v0.AuditEvidenceRetentionReviewRequest"
	req.SchemaVersion = "0.1.0"
	req.RequestID = defaultRequestID()

	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.AuditEvidenceRetentionReview(ctx, req)
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleAuditEvidenceBundleManifestGet(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("audit-evidence-bundle-manifest-get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	requestFile := fs.String("request-file", "", "path to bundle manifest request JSON")
	externalSharing := fs.Bool("external-sharing", false, "sign and return a signed manifest envelope for external sharing")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "audit-evidence-bundle-manifest-get usage: runecode-broker audit-evidence-bundle-manifest-get --request-file path [--external-sharing]"}
	}
	if strings.TrimSpace(*requestFile) == "" {
		return &usageError{message: "audit-evidence-bundle-manifest-get requires --request-file"}
	}
	if len(fs.Args()) != 0 {
		return &usageError{message: "audit-evidence-bundle-manifest-get does not accept positional arguments"}
	}
	req := brokerapi.AuditEvidenceBundleManifestGetRequest{}
	if err := loadStrictJSONFileValue(strings.TrimSpace(*requestFile), &req); err != nil {
		return err
	}
	req.SchemaID = "runecode.protocol.v0.AuditEvidenceBundleManifestGetRequest"
	req.SchemaVersion = "0.1.0"
	req.RequestID = defaultRequestID()
	req.ExternalSharingIntended = *externalSharing

	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.AuditEvidenceBundleManifestGet(ctx, req)
	if errResp != nil {
		return localAPIError(errResp)
	}
	if resp.SignedManifest != nil {
		return writeJSON(stdout, map[string]any{"manifest": resp.Manifest, "signed_manifest": resp.SignedManifest})
	}
	return writeJSON(stdout, resp.Manifest)
}

func handleAuditEvidenceBundleExport(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("audit-evidence-bundle-export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	requestFile := fs.String("request-file", "", "path to bundle export request JSON")
	outPath := fs.String("out", "", "archive output file path")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "audit-evidence-bundle-export usage: runecode-broker audit-evidence-bundle-export --request-file path --out path"}
	}
	if strings.TrimSpace(*requestFile) == "" || strings.TrimSpace(*outPath) == "" {
		return &usageError{message: "audit-evidence-bundle-export requires --request-file and --out"}
	}
	if len(fs.Args()) != 0 {
		return &usageError{message: "audit-evidence-bundle-export does not accept positional arguments"}
	}
	req := brokerapi.AuditEvidenceBundleExportRequest{}
	if err := loadStrictJSONFileValue(strings.TrimSpace(*requestFile), &req); err != nil {
		return err
	}
	req.SchemaID = "runecode.protocol.v0.AuditEvidenceBundleExportRequest"
	req.SchemaVersion = "0.1.0"
	req.RequestID = defaultRequestID()

	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	events, errResp := api.AuditEvidenceBundleExport(ctx, req)
	if errResp != nil {
		return localAPIError(errResp)
	}
	written, manifest, err := writeAuditEvidenceBundleExportToFile(events, strings.TrimSpace(*outPath))
	if err != nil {
		return err
	}
	if manifest != nil {
		return writeJSON(stdout, map[string]any{"wrote_bytes": written, "out": strings.TrimSpace(*outPath), "manifest": manifest})
	}
	_, err = fmt.Fprintf(stdout, "wrote %d bytes to %s\n", written, strings.TrimSpace(*outPath))
	return err
}

func handleAuditEvidenceBundleOfflineVerify(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("audit-evidence-bundle-offline-verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	bundlePath := fs.String("bundle", "", "path to evidence bundle archive")
	archiveFormat := fs.String("archive-format", "tar", "bundle archive format")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "audit-evidence-bundle-offline-verify usage: runecode-broker audit-evidence-bundle-offline-verify --bundle path [--archive-format tar]"}
	}
	if strings.TrimSpace(*bundlePath) == "" {
		return &usageError{message: "audit-evidence-bundle-offline-verify requires --bundle"}
	}
	if len(fs.Args()) != 0 {
		return &usageError{message: "audit-evidence-bundle-offline-verify does not accept positional arguments"}
	}

	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.AuditEvidenceBundleOfflineVerify(ctx, brokerapi.AuditEvidenceBundleOfflineVerifyRequest{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleOfflineVerifyRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		BundlePath:    strings.TrimSpace(*bundlePath),
		ArchiveFormat: strings.TrimSpace(*archiveFormat),
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp.Verification)
}

func writeAuditEvidenceBundleExportToFile(events []brokerapi.AuditEvidenceBundleExportEvent, outPath string) (int64, *brokerapi.AuditEvidenceBundleManifest, error) {
	parentDir := filepath.Dir(outPath)
	f, err := os.CreateTemp(parentDir, filepath.Base(outPath)+".*.tmp")
	if err != nil {
		return 0, nil, err
	}
	tmpPath := f.Name()
	defer os.Remove(tmpPath)
	var written int64
	sawCompleted := false
	var manifest *brokerapi.AuditEvidenceBundleManifest
	for i := range events {
		writtenNow, nextManifest, completed, err := handleAuditEvidenceBundleExportFileEvent(f, events[i])
		if err != nil {
			_ = f.Close()
			return 0, nil, err
		}
		written += writtenNow
		if nextManifest != nil {
			manifest = nextManifest
		}
		sawCompleted = sawCompleted || completed
	}
	if !sawCompleted {
		_ = f.Close()
		return 0, nil, fmt.Errorf("audit evidence bundle export did not complete successfully")
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return 0, nil, err
	}
	if err := f.Close(); err != nil {
		return 0, nil, err
	}
	if err := replaceFile(tmpPath, outPath); err != nil {
		return 0, nil, err
	}
	return written, manifest, nil
}

func handleAuditEvidenceBundleExportFileEvent(file *os.File, event brokerapi.AuditEvidenceBundleExportEvent) (int64, *brokerapi.AuditEvidenceBundleManifest, bool, error) {
	switch event.EventType {
	case "audit_evidence_bundle_export_start":
		if event.Manifest == nil {
			return 0, nil, false, nil
		}
		manifest := *event.Manifest
		return 0, &manifest, false, nil
	case "audit_evidence_bundle_export_chunk":
		written, err := writeAuditEvidenceBundleExportChunk(file, event.ChunkBase64)
		return written, nil, false, err
	case "audit_evidence_bundle_export_terminal":
		return 0, nil, event.TerminalStatus == "completed", auditEvidenceBundleExportTerminalErr(event)
	default:
		return 0, nil, false, nil
	}
}

func writeAuditEvidenceBundleExportChunk(file *os.File, chunkBase64 string) (int64, error) {
	chunk, err := base64.StdEncoding.DecodeString(chunkBase64)
	if err != nil {
		return 0, err
	}
	written, err := file.Write(chunk)
	return int64(written), err
}

func auditEvidenceBundleExportTerminalErr(event brokerapi.AuditEvidenceBundleExportEvent) error {
	if event.Error == nil {
		return nil
	}
	return fmt.Errorf("%s: %s", event.Error.Code, event.Error.Message)
}
