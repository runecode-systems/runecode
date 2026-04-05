// Command runecode-broker provides a local artifact and policy broker surface.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type usageError struct{ message string }

func (e *usageError) Error() string { return e.message }

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		var u *usageError
		if errors.As(err, &u) {
			fmt.Fprintln(os.Stderr, u.Error())
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return writeHelp(stdout)
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		return writeHelp(stdout)
	}

	service, err := brokerServiceFactory()
	if err != nil {
		return fmt.Errorf("runecode-broker failed to initialize store: %w", err)
	}
	handler, ok := commandHandlers()[args[0]]
	if !ok {
		_ = writeHelp(stderr)
		return &usageError{message: fmt.Sprintf("unknown command %q", args[0])}
	}
	return handler(args[1:], service, stdout)
}

var brokerServiceFactory = brokerService

type commandHandler func([]string, *brokerapi.Service, io.Writer) error

func commandHandlers() map[string]commandHandler {
	return map[string]commandHandler{
		"list-artifacts":          handleListArtifacts,
		"head-artifact":           handleHeadArtifact,
		"get-artifact":            handleGetArtifact,
		"put-artifact":            handlePutArtifact,
		"check-flow":              handleCheckFlow,
		"promote-excerpt":         handlePromoteExcerpt,
		"revoke-approved-excerpt": handleRevokeApprovedExcerpt,
		"set-run-status":          handleSetRunStatus,
		"gc":                      handleGC,
		"export-backup":           handleExportBackup,
		"restore-backup":          handleRestoreBackup,
		"show-audit":              handleShowAudit,
		"show-policy":             handleShowPolicy,
		"set-reserved-classes":    handleSetReservedClasses,
	}
}

func handleListArtifacts(_ []string, service *brokerapi.Service, stdout io.Writer) error {
	return writeJSON(stdout, service.List())
}

func handleHeadArtifact(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("head-artifact", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	digest := fs.String("digest", "", "artifact digest")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "head-artifact usage: runecode-broker head-artifact --digest sha256:..."}
	}
	if *digest == "" {
		return &usageError{message: "head-artifact requires --digest"}
	}
	record, err := service.Head(*digest)
	if err != nil {
		return err
	}
	return writeJSON(stdout, record)
}

func handleGetArtifact(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("get-artifact", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	digest := fs.String("digest", "", "artifact digest")
	out := fs.String("out", "", "output file path")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "get-artifact usage: runecode-broker get-artifact --digest sha256:... --out path"}
	}
	if *digest == "" || *out == "" {
		return &usageError{message: "get-artifact requires --digest and --out"}
	}
	r, err := service.Get(*digest)
	if err != nil {
		return err
	}
	tmpPath := *out + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		_ = r.Close()
		return err
	}
	defer os.Remove(tmpPath)
	written, err := io.Copy(f, r)
	if err != nil {
		_ = r.Close()
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = r.Close()
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		_ = r.Close()
		return err
	}
	if err := r.Close(); err != nil {
		return err
	}
	if err := replaceFile(tmpPath, *out); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "wrote %d bytes to %s\n", written, *out)
	return err
}

func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if removeErr := os.Remove(dst); removeErr != nil && !os.IsNotExist(removeErr) {
		return removeErr
	}
	return os.Rename(src, dst)
}

func handlePutArtifact(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("put-artifact", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	filePath := fs.String("file", "", "artifact payload file")
	contentType := fs.String("content-type", "application/octet-stream", "artifact content type")
	dataClass := fs.String("data-class", string(artifacts.DataClassSpecText), "artifact data class")
	provenance := fs.String("provenance-hash", "", "provenance receipt hash")
	role := fs.String("role", "workspace", "producer role")
	runID := fs.String("run-id", "", "run id")
	stepID := fs.String("step-id", "", "step id")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "put-artifact usage: runecode-broker put-artifact --file path --content-type text/plain --data-class spec_text --provenance-hash sha256:..."}
	}
	if *filePath == "" || *provenance == "" {
		return &usageError{message: "put-artifact requires --file and --provenance-hash"}
	}
	payload, err := os.ReadFile(*filePath)
	if err != nil {
		return err
	}
	class, err := brokerapi.ParseDataClass(*dataClass)
	if err != nil {
		return &usageError{message: err.Error()}
	}
	ref, err := service.Put(artifacts.PutRequest{
		Payload:               payload,
		ContentType:           *contentType,
		DataClass:             class,
		ProvenanceReceiptHash: *provenance,
		CreatedByRole:         *role,
		TrustedSource:         false,
		RunID:                 *runID,
		StepID:                *stepID,
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, ref)
}

func handleCheckFlow(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("check-flow", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	producer := fs.String("producer", "", "producer role")
	consumer := fs.String("consumer", "", "consumer role")
	dataClass := fs.String("data-class", "", "data class")
	digest := fs.String("digest", "", "digest")
	isEgress := fs.Bool("egress", false, "egress flow")
	manifestOptIn := fs.Bool("manifest-opt-in", false, "manifest opted in for approved excerpts")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "check-flow usage: runecode-broker check-flow --producer workspace --consumer model_gateway --data-class spec_text --digest sha256:... [--egress] [--manifest-opt-in]"}
	}
	if *producer == "" || *consumer == "" || *dataClass == "" || *digest == "" {
		return &usageError{message: "check-flow requires --producer --consumer --data-class --digest"}
	}
	class, err := brokerapi.ParseDataClass(*dataClass)
	if err != nil {
		return &usageError{message: err.Error()}
	}
	if err := service.CheckFlow(artifacts.FlowCheckRequest{ProducerRole: *producer, ConsumerRole: *consumer, DataClass: class, Digest: *digest, IsEgress: *isEgress, ManifestOptIn: *manifestOptIn}); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "allowed")
	return err
}

func handlePromoteExcerpt(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("promote-excerpt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	unapprovedDigest := fs.String("unapproved-digest", "", "source unapproved artifact digest")
	approver := fs.String("approver", "", "human approver id")
	approvalEnvelopePath := fs.String("approval-envelope", "", "path to signed approval decision envelope JSON")
	repoPath := fs.String("repo-path", "", "repo path")
	commit := fs.String("commit", "", "commit hash")
	extractorVersion := fs.String("extractor-version", "", "extractor tool version")
	fullContentVisible := fs.Bool("full-content-visible", false, "approval view showed full content")
	explicitViewFull := fs.Bool("explicit-view-full", false, "explicit view-full affordance used")
	bulk := fs.Bool("bulk", false, "bulk promotion request")
	bulkApproved := fs.Bool("bulk-approved", false, "separate bulk approval confirmed")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "promote-excerpt usage: runecode-broker promote-excerpt --unapproved-digest sha256:... --approver user --approval-envelope approval.json --repo-path path --commit hash --extractor-version v1 --full-content-visible"}
	}
	if *unapprovedDigest == "" {
		return &usageError{message: "promote-excerpt requires --unapproved-digest"}
	}
	approvalEnvelope, err := loadSignedApprovalEnvelope(*approvalEnvelopePath)
	if err != nil {
		return &usageError{message: fmt.Sprintf("invalid --approval-envelope: %v", err)}
	}
	ref, err := service.PromoteApprovedExcerpt(artifacts.PromotionRequest{
		UnapprovedDigest:      *unapprovedDigest,
		Approver:              *approver,
		ApprovalDecision:      approvalEnvelope,
		RepoPath:              *repoPath,
		Commit:                *commit,
		ExtractorToolVersion:  *extractorVersion,
		FullContentVisible:    *fullContentVisible,
		ExplicitViewFull:      *explicitViewFull,
		BulkRequest:           *bulk,
		BulkApprovalConfirmed: *bulkApproved,
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, ref)
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

func writeHelp(w io.Writer) error {
	_, err := fmt.Fprintln(w, `Usage: runecode-broker <command> [flags]

Commands:
  list-artifacts
  head-artifact --digest sha256:...
  get-artifact --digest sha256:... --out path
  put-artifact --file path --content-type type --data-class class --provenance-hash sha256:...
  check-flow --producer role --consumer role --data-class class --digest sha256:... [--egress] [--manifest-opt-in]
  promote-excerpt --unapproved-digest sha256:... --approver user --approval-envelope approval.json --repo-path path --commit hash --extractor-version v1 --full-content-visible
  revoke-approved-excerpt --digest sha256:... --actor user
  set-run-status --run-id id --status active|retained|closed
  gc
  export-backup --path backup.json
  restore-backup --path backup.json
  show-audit
  show-policy
  set-reserved-classes --enabled=true|false`)
	return err
}

func brokerService() (*brokerapi.Service, error) {
	return brokerapi.NewService(defaultBrokerStoreRoot())
}

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
