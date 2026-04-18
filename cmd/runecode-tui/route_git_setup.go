package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type gitSetupLoadedMsg struct {
	resp brokerapi.GitSetupGetResponse
	err  error
	seq  uint64
}

type gitSetupAuthBootstrapMsg struct {
	resp brokerapi.GitSetupAuthBootstrapResponse
	err  error
}

type gitSetupIdentityUpsertMsg struct {
	resp brokerapi.GitSetupIdentityUpsertResponse
	err  error
}

type gitSetupRouteModel struct {
	def               routeDefinition
	client            localBrokerClient
	loading           bool
	authBootstrapping bool
	upsertingIdentity bool
	errText           string
	status            string
	provider          string
	loadSeq           uint64
	data              brokerapi.GitSetupGetResponse
}

func newGitSetupRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return gitSetupRouteModel{def: def, client: client, provider: "github"}
}

func (m gitSetupRouteModel) ID() routeID { return m.def.ID }

func (m gitSetupRouteModel) Title() string { return m.def.Label }

func (m gitSetupRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		return m.handleRouteActivated(typed)
	case tea.KeyMsg:
		return m.handleKey(typed.String())
	case gitSetupLoadedMsg:
		return m.handleLoaded(typed)
	case gitSetupAuthBootstrapMsg:
		return m.handleAuthBootstrap(typed)
	case gitSetupIdentityUpsertMsg:
		return m.handleIdentityUpsert(typed)
	default:
		return m, nil
	}
}

func (m gitSetupRouteModel) handleRouteActivated(msg routeActivatedMsg) (routeModel, tea.Cmd) {
	if msg.RouteID != m.def.ID {
		return m, nil
	}
	m = m.beginLoad()
	return m, m.loadCmd(m.loadSeq)
}

func (m gitSetupRouteModel) handleKey(key string) (routeModel, tea.Cmd) {
	switch key {
	case "r":
		m = m.beginLoad()
		return m, m.loadCmd(m.loadSeq)
	case "a":
		return m.beginBootstrap(m.authBootstrapCmd)
	case "d":
		return m.beginBootstrap(m.authBootstrapDeviceCodeCmd)
	case "i":
		return m.beginIdentityUpsert()
	default:
		return m, nil
	}
}

func (m gitSetupRouteModel) beginBootstrap(cmd func() tea.Cmd) (routeModel, tea.Cmd) {
	if m.busy() {
		return m, nil
	}
	m.authBootstrapping = true
	m.errText = ""
	m.status = ""
	return m, cmd()
}

func (m gitSetupRouteModel) beginIdentityUpsert() (routeModel, tea.Cmd) {
	if m.busy() {
		return m, nil
	}
	m.upsertingIdentity = true
	m.errText = ""
	m.status = ""
	return m, m.identityUpsertCmd()
}

func (m gitSetupRouteModel) handleLoaded(msg gitSetupLoadedMsg) (routeModel, tea.Cmd) {
	if msg.seq != m.loadSeq {
		return m, nil
	}
	m.loading = false
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		return m, nil
	}
	m.errText = ""
	m.data = msg.resp
	if m.status == "" {
		m.status = "Press a (browser auth), d (device-code auth), i (identity profile upsert)."
	}
	return m, nil
}

func (m gitSetupRouteModel) handleAuthBootstrap(msg gitSetupAuthBootstrapMsg) (routeModel, tea.Cmd) {
	m.authBootstrapping = false
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		m.status = ""
		return m, nil
	}
	m.errText = ""
	m.status = gitSetupBootstrapStatus(msg.resp)
	m = m.beginLoad()
	return m, m.loadCmd(m.loadSeq)
}

func (m gitSetupRouteModel) handleIdentityUpsert(msg gitSetupIdentityUpsertMsg) (routeModel, tea.Cmd) {
	m.upsertingIdentity = false
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		m.status = ""
		return m, nil
	}
	m.errText = ""
	m.status = fmt.Sprintf("Identity profile %q upserted as broker-managed configuration.", msg.resp.Profile.ProfileID)
	m = m.beginLoad()
	return m, m.loadCmd(m.loadSeq)
}

func (m gitSetupRouteModel) busy() bool {
	return m.loading || m.authBootstrapping || m.upsertingIdentity
}

func gitSetupBootstrapStatus(resp brokerapi.GitSetupAuthBootstrapResponse) string {
	if resp.Status == "pending" && strings.TrimSpace(resp.DeviceVerificationURI) != "" {
		return fmt.Sprintf("Device-code bootstrap pending. Open %s and enter code %s.", resp.DeviceVerificationURI, resp.DeviceUserCode)
	}
	return fmt.Sprintf("Provider auth bootstrap status=%s mode=%s", resp.Status, resp.Mode)
}

func (m gitSetupRouteModel) beginLoad() gitSetupRouteModel {
	m.loading = true
	m.errText = ""
	m.loadSeq++
	return m
}

func (m gitSetupRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return renderStateCard(routeLoadStateLoading, "Git Setup", "Loading broker-owned git setup state...")
	}
	if m.authBootstrapping {
		return renderStateCard(routeLoadStateLoading, "Git Setup", "Bootstrapping provider auth using broker typed flow...")
	}
	if m.upsertingIdentity {
		return renderStateCard(routeLoadStateLoading, "Git Setup", "Saving commit identity profile through broker typed flow...")
	}
	if m.errText != "" {
		return renderStateCard(routeLoadStateError, "Git Setup", "Load failed: "+m.errText+" (press r to retry)")
	}
	account := m.data.ProviderAccount
	auth := m.data.AuthPosture
	control := m.data.ControlPlaneState
	profiles := m.data.IdentityProfiles
	profileSummary := "none"
	if len(profiles) > 0 {
		ids := make([]string, 0, len(profiles))
		for _, p := range profiles {
			ids = append(ids, p.ProfileID)
		}
		profileSummary = strings.Join(ids, ", ")
	}
	return compactLines(
		sectionTitle("Git Setup")+" "+focusBadge(focus),
		"Broker-owned setup/config state (non-policy authority):",
		fmt.Sprintf("Provider account: provider=%s linked=%t account=%s", valueOrNA(account.Provider), account.Linked, valueOrNA(account.AccountUsername)),
		fmt.Sprintf("Auth posture: status=%s bootstrap_mode=%s headless_supported=%t interactive_token_fallback=%t", valueOrNA(auth.AuthStatus), valueOrNA(auth.BootstrapMode), auth.HeadlessBootstrapSupported, auth.InteractiveTokenFallbackSupport),
		fmt.Sprintf("Commit identity profiles: count=%d profiles=%s default=%s", len(profiles), profileSummary, valueOrNA(control.DefaultIdentityProfileID)),
		fmt.Sprintf("Control-plane convenience state: last_view=%s recent_repositories=%d", valueOrNA(control.LastSetupView), len(control.RecentRepositories)),
		fmt.Sprintf("Policy authority: artifact_managed_only=%t inspect_supported=%t prepare_changes_supported=%t direct_mutation_supported=%t", m.data.PolicySurface.ArtifactManagedOnly, m.data.PolicySurface.InspectionSupported, m.data.PolicySurface.PrepareChangesSupport, m.data.PolicySurface.DirectMutationSupport),
		"Policy edits must stay artifact-managed. This route only inspects and prepares setup via broker APIs.",
		m.status,
		keyHint("Route keys: r reload, a browser auth, d device-code auth, i identity upsert"),
	)
}

func (m gitSetupRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	mainWidth := routeRegionWidth(ctx.Regions.Main, ctx.Width)
	mainHeight := routeRegionHeight(ctx.Regions.Main, ctx.Height)
	status := strings.TrimSpace(m.status)
	if status == "" && strings.TrimSpace(m.errText) != "" {
		status = "Load failed: " + strings.TrimSpace(m.errText)
	}
	return routeSurface{
		Regions: routeSurfaceRegions{
			Main:   routeSurfaceRegion{Title: "Git setup", Body: m.View(mainWidth, mainHeight, ctx.Focus)},
			Bottom: routeSurfaceRegion{Body: keyHint("Route keys: r reload, a browser auth, d device-code auth, i identity upsert")},
			Status: routeSurfaceRegion{Body: status},
		},
		Capabilities: routeSurfaceCapabilities{},
		Chrome:       routeSurfaceChrome{Breadcrumbs: []string{"Home", m.def.Label}},
	}
}

func (m gitSetupRouteModel) loadCmd(seq uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.GitSetupGet(ctx, m.provider)
		if err != nil {
			return gitSetupLoadedMsg{err: err, seq: seq}
		}
		return gitSetupLoadedMsg{resp: resp, seq: seq}
	}
}

func (m gitSetupRouteModel) authBootstrapCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.GitSetupAuthBootstrap(ctx, brokerapi.GitSetupAuthBootstrapRequest{Provider: m.provider, Mode: "browser"})
		if err != nil {
			return gitSetupAuthBootstrapMsg{err: err}
		}
		return gitSetupAuthBootstrapMsg{resp: resp}
	}
}

func (m gitSetupRouteModel) authBootstrapDeviceCodeCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.GitSetupAuthBootstrap(ctx, brokerapi.GitSetupAuthBootstrapRequest{Provider: m.provider, Mode: "device_code"})
		if err != nil {
			return gitSetupAuthBootstrapMsg{err: err}
		}
		return gitSetupAuthBootstrapMsg{resp: resp}
	}
}

func (m gitSetupRouteModel) identityUpsertCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.GitSetupIdentityUpsert(ctx, brokerapi.GitSetupIdentityUpsertRequest{Provider: m.provider, Profile: brokerapi.GitCommitIdentityProfile{ProfileID: "default", DisplayName: "Default identity", AuthorName: "RuneCode Operator", AuthorEmail: "operator@example.invalid", CommitterName: "RuneCode Operator", CommitterEmail: "operator@example.invalid", SignoffName: "RuneCode Operator", SignoffEmail: "operator@example.invalid", DefaultProfile: true}})
		if err != nil {
			return gitSetupIdentityUpsertMsg{err: err}
		}
		return gitSetupIdentityUpsertMsg{resp: resp}
	}
}
