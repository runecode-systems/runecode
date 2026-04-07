package brokerapi

import (
	"fmt"
	"io"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type Service struct {
	store       *artifacts.Store
	auditLedger *auditd.Ledger
	apiConfig   APIConfig
	apiInflight *inFlightGate
	approvals   approvalState
	versionInfo BrokerVersionInfo
}

func NewService(storeRoot string, ledgerRoot string) (*Service, error) {
	return NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{})
}

func NewServiceWithConfig(storeRoot string, ledgerRoot string, cfg APIConfig) (*Service, error) {
	store, err := artifacts.NewStore(storeRoot)
	if err != nil {
		return nil, err
	}
	ledger, err := auditd.Open(ledgerRoot)
	if err != nil {
		return nil, err
	}
	resolved := cfg.withDefaults()
	return &Service{
		store:       store,
		auditLedger: ledger,
		apiConfig:   resolved,
		apiInflight: newInFlightGate(resolved.Limits),
		approvals:   approvalState{records: map[string]approvalRecord{}},
		versionInfo: BrokerVersionInfo{
			SchemaID:                    "runecode.protocol.v0.BrokerVersionInfo",
			SchemaVersion:               "0.1.0",
			ProductVersion:              "0.0.0-dev",
			BuildRevision:               "unknown",
			BuildTime:                   "unknown",
			ProtocolBundleVersion:       "0.5.0",
			ProtocolBundleManifestHash:  "sha256:" + strings.Repeat("0", 64),
			APIFamily:                   "broker_local_api",
			APIVersion:                  "v0",
			SupportedTransportEncodings: []string{"json"},
		},
	}, nil
}

func (s *Service) Put(req artifacts.PutRequest) (artifacts.ArtifactReference, error) {
	return s.store.Put(req)
}

func (s *Service) List() []artifacts.ArtifactRecord {
	return s.store.List()
}

func (s *Service) RunStatuses() map[string]string {
	return s.store.RunStatuses()
}

func (s *Service) Head(digest string) (artifacts.ArtifactRecord, error) {
	return s.store.Head(digest)
}

func (s *Service) Get(digest string) (io.ReadCloser, error) {
	return s.store.Get(digest)
}

func (s *Service) GetForFlow(req artifacts.ArtifactReadRequest) (io.ReadCloser, artifacts.ArtifactRecord, error) {
	return s.store.GetForFlow(req)
}

func (s *Service) CheckFlow(req artifacts.FlowCheckRequest) error {
	return s.store.CheckFlow(req)
}

func (s *Service) PromoteApprovedExcerpt(req artifacts.PromotionRequest) (artifacts.ArtifactReference, error) {
	return s.store.PromoteApprovedExcerpt(req)
}

func (s *Service) RevokeApprovedExcerpt(digest, actor string) error {
	return s.store.RevokeApprovedExcerpt(digest, actor)
}

func (s *Service) SetRunStatus(runID, status string) error {
	return s.store.SetRunStatus(runID, status)
}

func (s *Service) GarbageCollect() (artifacts.GCResult, error) {
	return s.store.GarbageCollect()
}

func (s *Service) ExportBackup(path string) error {
	return s.store.ExportBackup(path)
}

func (s *Service) RestoreBackup(path string) error {
	return s.store.RestoreBackup(path)
}

func (s *Service) ReadAuditEvents() ([]artifacts.AuditEvent, error) {
	return s.store.ReadAuditEvents()
}

func (s *Service) SetPolicy(policy artifacts.Policy) error {
	return s.store.SetPolicy(policy)
}

func (s *Service) Policy() artifacts.Policy {
	return s.store.Policy()
}

func (s *Service) AuditReadiness() (trustpolicy.AuditdReadiness, error) {
	return s.auditLedger.Readiness()
}

type AuditVerificationSurface struct {
	Summary trustpolicy.DerivedRunAuditVerificationSummary `json:"summary"`
	Report  trustpolicy.AuditVerificationReportPayload     `json:"report"`
	Views   []trustpolicy.AuditOperationalView             `json:"views"`
}

func (s *Service) LatestAuditVerificationSurface(limit int) (AuditVerificationSurface, error) {
	summary, views, report, err := s.auditLedger.LatestVerificationSummaryAndViews(limit)
	if err != nil {
		return AuditVerificationSurface{}, err
	}
	return AuditVerificationSurface{Summary: summary, Report: report, Views: views}, nil
}

func (s *Service) APILimits() Limits {
	return s.apiConfig.Limits
}

func ParseDataClass(value string) (artifacts.DataClass, error) {
	class := artifacts.DataClass(value)
	switch class {
	case artifacts.DataClassSpecText,
		artifacts.DataClassUnapprovedFileExcerpts,
		artifacts.DataClassApprovedFileExcerpts,
		artifacts.DataClassDiffs,
		artifacts.DataClassBuildLogs,
		artifacts.DataClassAuditEvents,
		artifacts.DataClassAuditVerificationReport,
		artifacts.DataClassWebQuery,
		artifacts.DataClassWebCitations:
		return class, nil
	default:
		return "", fmt.Errorf("unsupported data class %q", value)
	}
}
