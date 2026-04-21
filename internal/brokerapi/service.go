package brokerapi

import (
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/projectsubstrate"
	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

var (
	BrokerProductVersion = "0.0.0-dev"
	BrokerBuildRevision  = "dev"
	BrokerBuildTime      = "1970-01-01T00:00:00Z"
)

const (
	brokerProtocolBundleVersion      = "0.9.0"
	brokerProtocolBundleManifestHash = "sha256:47427e96642a0f2cb7fb4e66aed61817f72f4233f0273744baa8469a2d13f170"
)

type Service struct {
	store                      *artifacts.Store
	auditLedger                *auditd.Ledger
	secretsSvc                 *secretsd.Service
	gitMutationExecutor        gitRemoteMutationExecutor
	auditor                    *brokerAuditEmitter
	auditRoot                  string
	gatewayQuota               *gatewayQuotaBackend
	gatewayRuntime             *modelGatewayRuntime
	providerSubstrate          *providerSubstrateState
	providerSetup              *providerSetupState
	instancePostureController  instanceBackendPostureController
	apiConfig                  APIConfig
	apiInflight                *inFlightGate
	versionInfo                BrokerVersionInfo
	gitSetup                   *gitSetupState
	projectSubstrate           projectsubstrate.DiscoveryResult
	discoverProjectSubstrateFn func() (projectsubstrate.DiscoveryResult, error)
	now                        func() time.Time
}

func NewService(storeRoot string, ledgerRoot string) (*Service, error) {
	return NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{})
}

func NewServiceWithConfig(storeRoot string, ledgerRoot string, cfg APIConfig) (*Service, error) {
	resolved := cfg.withDefaults()
	store, ledger, auditor, err := newServiceDependencies(storeRoot, ledgerRoot)
	if err != nil {
		return nil, err
	}
	svc := newConfiguredService(store, ledger, ledgerRoot, auditor, resolved)
	if secretsSvc, secretsErr := openLocalSecretsService(); secretsErr == nil {
		svc.secretsSvc = secretsSvc
	}
	svc.providerSetup = newProviderSetupState(time.Now)
	authority := projectsubstrate.RepoRootAuthorityProcessWorkingDirectory
	if resolved.RepositoryRoot != "" {
		authority = projectsubstrate.RepoRootAuthorityExplicitConfig
	}
	projectSubstrateResult, err := projectsubstrate.DiscoverAndValidate(projectsubstrate.DiscoveryInput{
		RepositoryRoot: resolved.RepositoryRoot,
		Authority:      authority,
	})
	if err != nil {
		return nil, err
	}
	svc.projectSubstrate = projectSubstrateResult
	svc.configureProviderDurability()
	if err := svc.reloadProviderDurableState(); err != nil {
		return nil, err
	}
	return svc, nil
}

func newServiceDependencies(storeRoot, ledgerRoot string) (*artifacts.Store, *auditd.Ledger, *brokerAuditEmitter, error) {
	store, err := artifacts.NewStore(storeRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	ledger, err := auditd.Open(ledgerRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	auditor, err := newBrokerAuditEmitter()
	if err != nil {
		return nil, nil, nil, err
	}
	return store, ledger, auditor, nil
}

func newConfiguredService(store *artifacts.Store, ledger *auditd.Ledger, ledgerRoot string, auditor *brokerAuditEmitter, cfg APIConfig) *Service {
	quotaBackend := newGatewayQuotaBackend()
	quotaBackend.setLimits(cfg.GatewayQuota)
	runtime := newModelGatewayRuntime(quotaBackend)
	svc := &Service{
		store:                     store,
		auditLedger:               ledger,
		auditor:                   auditor,
		auditRoot:                 ledgerRoot,
		gatewayQuota:              quotaBackend,
		gatewayRuntime:            runtime,
		providerSubstrate:         newProviderSubstrateState(time.Now),
		instancePostureController: newLocalInstanceBackendPostureController(),
		gitSetup:                  newGitSetupState(),
		apiConfig:                 cfg,
		apiInflight:               newInFlightGate(cfg.Limits),
		now:                       time.Now,
		versionInfo:               defaultBrokerVersionInfo(),
	}
	runtime.auditFn = svc.AppendTrustedAuditEvent
	return svc
}

func openLocalSecretsService() (*secretsd.Service, error) {
	root, err := validatedSecretsStateRoot(defaultSecretsStateRoot())
	if err != nil {
		return nil, err
	}
	return secretsd.Open(root)
}

func defaultBrokerVersionInfo() BrokerVersionInfo {
	return BrokerVersionInfo{
		SchemaID:                        "runecode.protocol.v0.BrokerVersionInfo",
		SchemaVersion:                   "0.1.0",
		ProductVersion:                  BrokerProductVersion,
		BuildRevision:                   BrokerBuildRevision,
		BuildTime:                       BrokerBuildTime,
		ProtocolBundleVersion:           brokerProtocolBundleVersion,
		ProtocolBundleManifestHash:      brokerProtocolBundleManifestHash,
		APIFamily:                       "broker_local_api",
		APIVersion:                      "v0",
		SupportedTransportEncodings:     []string{"json"},
		ProjectSubstrateContractID:      "",
		ProjectSubstrateContractVersion: "",
		ProjectSubstrateVersion:         "",
		ProjectSubstrateValidationState: "",
		ProjectSubstratePosture:         "",
		ProjectSubstrateBlockedReasons:  []string{},
		ProjectSubstrateSupportedMin:    "",
		ProjectSubstrateSupportedMax:    "",
		ProjectSubstrateRecommended:     "",
		ProjectContextIdentityDigest:    "",
		ProjectSubstratePostureSummary:  nil,
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

func (s *Service) SetNowFuncForTests(nowFn func() time.Time) {
	if nowFn == nil {
		s.now = time.Now
		s.store.SetNowFuncForTests(nil)
		return
	}
	s.now = nowFn
	s.store.SetNowFuncForTests(nowFn)
	if s.providerSubstrate != nil {
		s.providerSubstrate.setNowFunc(nowFn)
	}
	if s.providerSetup != nil {
		s.providerSetup.setNowFunc(nowFn)
	}
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

func (s *Service) APILimits() Limits { return s.apiConfig.Limits }

func (s *Service) discoverProjectSubstrate() (projectsubstrate.DiscoveryResult, error) {
	if s == nil {
		return projectsubstrate.DiscoveryResult{}, nil
	}
	if s.discoverProjectSubstrateFn != nil {
		result, err := s.discoverProjectSubstrateFn()
		if err != nil {
			return projectsubstrate.DiscoveryResult{}, err
		}
		s.projectSubstrate = result
		return result, nil
	}
	repoRoot := strings.TrimSpace(s.projectSubstrate.RepositoryRoot)
	if repoRoot == "" {
		repoRoot = strings.TrimSpace(s.apiConfig.RepositoryRoot)
	}
	result, err := projectsubstrate.DiscoverAndValidate(projectsubstrate.DiscoveryInput{
		RepositoryRoot: repoRoot,
		Authority:      s.projectSubstrateAuthority(),
	})
	if err != nil {
		return projectsubstrate.DiscoveryResult{}, err
	}
	s.projectSubstrate = result
	return result, nil
}
