package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func handlePromoteExcerpt(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("promote-excerpt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	unapprovedDigest := fs.String("unapproved-digest", "", "source unapproved artifact digest")
	approver := fs.String("approver", "", "human approver id")
	approvalRequestPath := fs.String("approval-request", "", "path to signed approval request envelope JSON")
	approvalEnvelopePath := fs.String("approval-envelope", "", "path to signed approval decision envelope JSON")
	repoPath := fs.String("repo-path", "", "repo path")
	commit := fs.String("commit", "", "commit hash")
	extractorVersion := fs.String("extractor-version", "", "extractor tool version")
	fullContentVisible := fs.Bool("full-content-visible", false, "approval view showed full content")
	explicitViewFull := fs.Bool("explicit-view-full", false, "explicit view-full affordance used")
	bulk := fs.Bool("bulk", false, "bulk promotion request")
	bulkApproved := fs.Bool("bulk-approved", false, "separate bulk approval confirmed")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "promote-excerpt usage: runecode-broker promote-excerpt --unapproved-digest sha256:... --approver user --approval-request approval-request.json --approval-envelope approval.json --repo-path path --commit hash --extractor-version v1 --full-content-visible"}
	}
	if *unapprovedDigest == "" {
		return &usageError{message: "promote-excerpt requires --unapproved-digest"}
	}
	approvalRequest, err := loadSignedApprovalEnvelope(*approvalRequestPath)
	if err != nil {
		return &usageError{message: fmt.Sprintf("invalid --approval-request: %v", err)}
	}
	approvalEnvelope, err := loadSignedApprovalEnvelope(*approvalEnvelopePath)
	if err != nil {
		return &usageError{message: fmt.Sprintf("invalid --approval-envelope: %v", err)}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resolveResp, errResp := api.ApprovalResolve(ctx, brokerapi.ApprovalResolveRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalResolveRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		BoundScope: brokerapi.ApprovalBoundScope{
			SchemaID:      "runecode.protocol.v0.ApprovalBoundScope",
			SchemaVersion: "0.1.0",
			ActionKind:    "promotion",
		},
		UnapprovedDigest:       *unapprovedDigest,
		Approver:               *approver,
		RepoPath:               *repoPath,
		Commit:                 *commit,
		ExtractorToolVersion:   *extractorVersion,
		FullContentVisible:     *fullContentVisible,
		ExplicitViewFull:       *explicitViewFull,
		BulkRequest:            *bulk,
		BulkApprovalConfirmed:  *bulkApproved,
		SignedApprovalRequest:  *approvalRequest,
		SignedApprovalDecision: *approvalEnvelope,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	if resolveResp.ApprovedArtifact == nil {
		return fmt.Errorf("gateway_failure: approval resolved without approved artifact")
	}
	return writeJSON(stdout, resolveResp.ApprovedArtifact.Reference)
}

func handleRevokeApprovedExcerpt(args []string, service *brokerapi.Service, _ io.Writer) error {
	fs := flag.NewFlagSet("revoke-approved-excerpt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	digest := fs.String("digest", "", "approved digest")
	actor := fs.String("actor", "", "actor")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "revoke-approved-excerpt usage: runecode-broker revoke-approved-excerpt --digest sha256:... --actor system"}
	}
	if *digest == "" || *actor == "" {
		return &usageError{message: "revoke-approved-excerpt requires --digest and --actor"}
	}
	return service.RevokeApprovedExcerpt(*digest, *actor)
}

func handleSetRunStatus(args []string, service *brokerapi.Service, _ io.Writer) error {
	fs := flag.NewFlagSet("set-run-status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	runID := fs.String("run-id", "", "run id")
	status := fs.String("status", "", "active|retained|closed")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "set-run-status usage: runecode-broker set-run-status --run-id run-1 --status retained"}
	}
	if *runID == "" || *status == "" {
		return &usageError{message: "set-run-status requires --run-id and --status"}
	}
	return service.SetRunStatus(*runID, *status)
}

func handleGC(_ []string, service *brokerapi.Service, stdout io.Writer) error {
	result, err := service.GarbageCollect()
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
}

func handleExportBackup(args []string, service *brokerapi.Service, _ io.Writer) error {
	fs := flag.NewFlagSet("export-backup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "output backup path")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "export-backup usage: runecode-broker export-backup --path backup.json"}
	}
	if *path == "" {
		return &usageError{message: "export-backup requires --path"}
	}
	return service.ExportBackup(*path)
}

func handleRestoreBackup(args []string, service *brokerapi.Service, _ io.Writer) error {
	fs := flag.NewFlagSet("restore-backup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "backup path")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "restore-backup usage: runecode-broker restore-backup --path backup.json"}
	}
	if *path == "" {
		return &usageError{message: "restore-backup requires --path"}
	}
	return service.RestoreBackup(*path)
}

func handleShowAudit(_ []string, service *brokerapi.Service, stdout io.Writer) error {
	events, err := service.ReadAuditEvents()
	if err != nil {
		return err
	}
	return writeJSON(stdout, events)
}

func handleShowPolicy(_ []string, service *brokerapi.Service, stdout io.Writer) error {
	return writeJSON(stdout, service.Policy())
}

func handleSetReservedClasses(args []string, service *brokerapi.Service, _ io.Writer) error {
	fs := flag.NewFlagSet("set-reserved-classes", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	enabled := fs.Bool("enabled", false, "enable reserved web_* data classes")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "set-reserved-classes usage: runecode-broker set-reserved-classes --enabled=true"}
	}
	policy := service.Policy()
	policy.ReservedClassesEnabled = *enabled
	return service.SetPolicy(policy)
}

func handleAuditReadiness(_ []string, service *brokerapi.Service, stdout io.Writer) error {
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ReadinessGet(ctx, brokerapi.ReadinessGetRequest{
		SchemaID:      "runecode.protocol.v0.ReadinessGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp.Readiness)
}

func handleAuditVerification(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("audit-verification", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	limit := fs.Int("limit", 20, "max operational view entries")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "audit-verification usage: runecode-broker audit-verification [--limit N]"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.AuditVerificationGet(ctx, brokerapi.AuditVerificationGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditVerificationGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		ViewLimit:     *limit,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, brokerapi.AuditVerificationSurface{Summary: resp.Summary, Report: resp.Report, Views: resp.Views})
}

func handleImportTrustedContract(args []string, service *brokerapi.Service, _ io.Writer) error {
	fs := flag.NewFlagSet("import-trusted-contract", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	kind := fs.String("kind", "", "trusted contract kind")
	filePath := fs.String("file", "", "path to contract JSON")
	evidencePath := fs.String("evidence", "", "path to trusted import evidence JSON")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "import-trusted-contract usage: runecode-broker import-trusted-contract --kind verifier-record --file verifier.json --evidence import-evidence.json"}
	}
	if *kind == "" || *filePath == "" || *evidencePath == "" {
		return &usageError{message: "import-trusted-contract requires --kind, --file, and --evidence"}
	}
	switch *kind {
	case "verifier-record":
		record, err := loadVerifierRecord(*filePath)
		if err != nil {
			return fmt.Errorf("invalid verifier record: %w", err)
		}
		evidence, err := loadTrustedImportRequest(*evidencePath)
		if err != nil {
			return fmt.Errorf("invalid import evidence: %w", err)
		}
		return putTrustedVerifierRecord(service, record, evidence)
	default:
		return &usageError{message: fmt.Sprintf("unsupported --kind %q (supported: verifier-record)", *kind)}
	}
}
