package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-systems/runecode/internal/brokerapi"
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
	if resp.Status == "pending" {
		return "Browser auth started. Finish sign-in with the provider, then reload if the linked account does not appear."
	}
	return fmt.Sprintf("Provider sign-in status: %s.", valueOrNA(resp.Status))
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
	return compactLines(
		sectionTitle("Git Setup")+" "+focusBadge(focus),
		renderStateCardSpec(gitSetupStateCard(account, auth, profiles)),
		fmt.Sprintf("Provider link: %s", gitProviderAccountSummary(account, auth)),
		fmt.Sprintf("Commit identity: %s", gitIdentitySummary(profiles, control.DefaultIdentityProfileID)),
		fmt.Sprintf("Review posture: %s", gitPolicySafetySummary(m.data.PolicySurface)),
		fmt.Sprintf("Next safe action: %s", gitNextSafeAction(account, profiles)),
		m.status,
	)
}

func gitSetupStateCard(account brokerapi.GitProviderAccountState, auth brokerapi.GitAuthPostureState, profiles []brokerapi.GitCommitIdentityProfile) stateCardSpec {
	state := routeLoadStateWaiting
	message := "Link a provider account before starting broker-managed review work."
	next := "Start browser auth or use device code to connect the provider account."
	if account.Linked && len(profiles) > 0 {
		state = routeLoadStateReady
		message = "Git account and commit identity are ready for broker-managed review flows."
		next = "Continue to review flows, or reload here if provider state changed."
	} else if account.Linked {
		message = "The provider account is linked, but commit identity still needs setup."
		next = "Upsert the default broker-managed commit identity before review work."
	} else if strings.TrimSpace(auth.AuthStatus) == "pending" {
		message = "Provider sign-in is in progress; the account link is not ready yet."
		next = "Finish the current sign-in step, then reload to confirm the linked account."
	}
	return stateCardSpec{State: state, Title: "Git setup", Message: message, Reason: "RuneCode needs a linked account and ready commit identity before broker-managed review work can proceed.", NextAction: next, ShortcutCue: "browser auth • device auth • identity upsert", EvidenceCue: "linked account and commit identity posture"}
}

func gitProviderAccountSummary(account brokerapi.GitProviderAccountState, auth brokerapi.GitAuthPostureState) string {
	if account.Linked {
		return fmt.Sprintf("%s is linked as %s.", valueOrNA(account.Provider), valueOrNA(account.AccountUsername))
	}
	if strings.TrimSpace(auth.AuthStatus) == "pending" {
		return fmt.Sprintf("%s sign-in is in progress.", valueOrNA(account.Provider))
	}
	return fmt.Sprintf("%s is not linked yet.", valueOrNA(account.Provider))
}

func gitIdentitySummary(profiles []brokerapi.GitCommitIdentityProfile, defaultID string) string {
	if len(profiles) == 0 {
		return "No commit identity is ready yet."
	}
	if strings.TrimSpace(defaultID) != "" {
		return fmt.Sprintf("Default commit identity %s is ready.", valueOrNA(defaultID))
	}
	return fmt.Sprintf("%d commit identity profile(s) are available.", len(profiles))
}

func gitPolicySafetySummary(policy brokerapi.GitPolicySurfaceState) string {
	if policy.ArtifactManagedOnly && !policy.DirectMutationSupport {
		return "Reviews stay artifact-managed and direct remote mutation remains disabled."
	}
	if policy.ArtifactManagedOnly {
		return "Reviews stay artifact-managed; any remote mutation still requires broker-gated policy checks."
	}
	if !policy.DirectMutationSupport {
		return "Direct remote mutation is disabled until broker policy allows it."
	}
	return "Remote review actions remain broker-gated by the current policy posture."
}

func gitNextSafeAction(account brokerapi.GitProviderAccountState, profiles []brokerapi.GitCommitIdentityProfile) string {
	if !account.Linked {
		return "Link the provider account with browser auth or device code."
	}
	if len(profiles) == 0 {
		return "Upsert the default broker-managed commit identity."
	}
	return "Move on to review flows, and reload here if provider state changes."
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
