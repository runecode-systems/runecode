package brokerapi

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

var (
	BrokerProductVersion = "0.0.0-dev"
	BrokerBuildRevision  = "dev"
	BrokerBuildTime      = "1970-01-01T00:00:00Z"
)

const (
	brokerProtocolBundleVersion      = "0.5.0"
	brokerProtocolBundleManifestHash = "sha256:98d83c70b6948c654d0e23e556eb62ab7a0cac54dc214ba755521d5002061b06"
)

type Service struct {
	store       *artifacts.Store
	auditLedger *auditd.Ledger
	auditor     *brokerAuditEmitter
	auditRoot   string
	apiConfig   APIConfig
	apiInflight *inFlightGate
	versionInfo BrokerVersionInfo
	now         func() time.Time

	runtimeFactsMu sync.RWMutex
	runtimeFacts   map[string]launcherbackend.RuntimeFactsSnapshot
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
	auditor, err := newBrokerAuditEmitter()
	if err != nil {
		return nil, err
	}
	return &Service{
		store:        store,
		auditLedger:  ledger,
		auditor:      auditor,
		auditRoot:    ledgerRoot,
		apiConfig:    resolved,
		apiInflight:  newInFlightGate(resolved.Limits),
		now:          time.Now,
		versionInfo:  defaultBrokerVersionInfo(),
		runtimeFacts: map[string]launcherbackend.RuntimeFactsSnapshot{},
	}, nil
}

func defaultBrokerVersionInfo() BrokerVersionInfo {
	return BrokerVersionInfo{
		SchemaID:                    "runecode.protocol.v0.BrokerVersionInfo",
		SchemaVersion:               "0.1.0",
		ProductVersion:              BrokerProductVersion,
		BuildRevision:               BrokerBuildRevision,
		BuildTime:                   BrokerBuildTime,
		ProtocolBundleVersion:       brokerProtocolBundleVersion,
		ProtocolBundleManifestHash:  brokerProtocolBundleManifestHash,
		APIFamily:                   "broker_local_api",
		APIVersion:                  "v0",
		SupportedTransportEncodings: []string{"json"},
	}
}

func (s *Service) SetVersionInfo(info BrokerVersionInfo) {
	if info.SchemaID == "" {
		info.SchemaID = "runecode.protocol.v0.BrokerVersionInfo"
	}
	if info.SchemaVersion == "" {
		info.SchemaVersion = "0.1.0"
	}
	if info.APIFamily == "" {
		info.APIFamily = "broker_local_api"
	}
	if info.APIVersion == "" {
		info.APIVersion = "v0"
	}
	if len(info.SupportedTransportEncodings) == 0 {
		info.SupportedTransportEncodings = []string{"json"}
	}
	s.versionInfo = info
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

func (s *Service) DeleteDigest(digest string) error {
	return s.store.DeleteDigest(digest)
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

func (s *Service) AppendTrustedAuditEvent(eventType, actor string, details map[string]interface{}) error {
	return s.store.AppendTrustedAuditEvent(eventType, actor, details)
}

func (s *Service) RecordPolicyDecision(runID string, digest string, decision policyengine.PolicyDecision) error {
	return s.store.RecordPolicyDecision(artifacts.PolicyDecisionRecord{
		Digest:                   digest,
		RunID:                    runID,
		SchemaID:                 decision.SchemaID,
		SchemaVersion:            decision.SchemaVersion,
		DecisionOutcome:          string(decision.DecisionOutcome),
		PolicyReasonCode:         decision.PolicyReasonCode,
		ManifestHash:             decision.ManifestHash,
		ActionRequestHash:        decision.ActionRequestHash,
		PolicyInputHashes:        append([]string{}, decision.PolicyInputHashes...),
		RelevantArtifactHashes:   append([]string{}, decision.RelevantArtifactHashes...),
		DetailsSchemaID:          decision.DetailsSchemaID,
		Details:                  decision.Details,
		RequiredApprovalSchemaID: decision.RequiredApprovalSchemaID,
		RequiredApproval:         decision.RequiredApproval,
	})
}

func (s *Service) PolicyDecisionRefsForRun(runID string) []string {
	return s.store.PolicyDecisionRefsForRun(runID)
}

func (s *Service) ApprovalList() []artifacts.ApprovalRecord {
	return s.store.ApprovalList()
}

func (s *Service) ApprovalGet(approvalID string) (artifacts.ApprovalRecord, bool) {
	return s.store.ApprovalGet(approvalID)
}

func (s *Service) RecordApproval(record artifacts.ApprovalRecord) error {
	return s.store.RecordApproval(record)
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
	if s.auditLedger == nil {
		return AuditVerificationSurface{}, fmt.Errorf("audit ledger unavailable")
	}
	summary, views, report, err := s.auditLedger.LatestVerificationSummaryAndViews(limit)
	if err != nil {
		return AuditVerificationSurface{}, err
	}
	return AuditVerificationSurface{Summary: summary, Report: report, Views: views}, nil
}

func (s *Service) APILimits() Limits {
	return s.apiConfig.Limits
}
