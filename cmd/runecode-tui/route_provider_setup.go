package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

type providerSetupRouteMsg struct {
	beginResp       *brokerapi.ProviderSetupSessionBeginResponse
	prepareResp     *brokerapi.ProviderSetupSecretIngressPrepareResponse
	submitResp      *brokerapi.ProviderSetupSecretIngressSubmitResponse
	profileListResp *brokerapi.ProviderProfileListResponse
	err             error
}

type providerSetupRouteModel struct {
	def         routeDefinition
	client      localBrokerClient
	selected    providerSetupDefaults
	status      string
	errText     string
	loading     bool
	entryActive bool
	secretRunes []rune
	begin       brokerapi.ProviderSetupSessionBeginResponse
	prepare     brokerapi.ProviderSetupSecretIngressPrepareResponse
	submit      brokerapi.ProviderSetupSecretIngressSubmitResponse
	profiles    []brokerapi.ProviderProfile
}

func newProviderSetupRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return providerSetupRouteModel{def: def, client: client, selected: providerSetupDefaultsFor("openai_compatible"), status: "Press s to start direct-credential setup. Press f to switch provider family."}
}

func (m providerSetupRouteModel) ID() routeID   { return m.def.ID }
func (m providerSetupRouteModel) Title() string { return m.def.Label }

func (m providerSetupRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		return m.handleRouteActivated(typed)
	case tea.KeyMsg:
		return m.handleKey(typed)
	case providerSetupRouteMsg:
		return m.handleRouteMsg(typed)
	default:
		return m, nil
	}
}

func (m providerSetupRouteModel) handleRouteActivated(msg routeActivatedMsg) (routeModel, tea.Cmd) {
	if msg.RouteID != m.def.ID {
		return m, nil
	}
	m.loading = true
	m.errText = ""
	return m, m.refreshProfilesCmd()
}

func (m providerSetupRouteModel) handleRouteMsg(msg providerSetupRouteMsg) (routeModel, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		return m, nil
	}
	m.errText = ""
	if msg.beginResp != nil {
		m.begin = *msg.beginResp
		m.status = "Session started. Type API credential (masked), Enter to submit, Esc to cancel."
		m.entryActive = true
		m.secretRunes = nil
	}
	if msg.prepareResp != nil {
		m.prepare = *msg.prepareResp
	}
	if msg.submitResp != nil {
		m.submit = *msg.submitResp
		m.status = fmt.Sprintf("Direct credential stored in secretsd for profile %s.", m.submit.Profile.ProviderProfileID)
		m.entryActive = false
		m.secretRunes = nil
		m.profiles = upsertProviderProfile(m.profiles, m.submit.Profile)
		m.loading = true
		return m, m.refreshProfilesCmd()
	}
	if msg.profileListResp != nil {
		m.profiles = append([]brokerapi.ProviderProfile{}, msg.profileListResp.Profiles...)
		if strings.TrimSpace(m.status) == "" {
			m.status = "Press s to start direct-credential setup."
		}
	}
	return m, nil
}

func (m providerSetupRouteModel) handleKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	if m.entryActive {
		return m.handleSecretEntryKey(key)
	}
	return m.handleIdleKey(key)
}

func (m providerSetupRouteModel) handleSecretEntryKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	switch key.Type {
	case tea.KeyEsc:
		m.entryActive = false
		m.secretRunes = nil
		m.status = "Secret entry cancelled. Press s to restart setup."
		return m, nil
	case tea.KeyBackspace, tea.KeyDelete:
		if n := len(m.secretRunes); n > 0 {
			m.secretRunes = m.secretRunes[:n-1]
		}
		return m, nil
	case tea.KeyEnter:
		if len(m.secretRunes) == 0 {
			m.status = "Secret cannot be empty."
			return m, nil
		}
		m.loading = true
		return m, m.submitSecretCmd([]byte(string(m.secretRunes)))
	case tea.KeyRunes:
		m.secretRunes = append(m.secretRunes, key.Runes...)
		return m, nil
	default:
		return m, nil
	}
}

func (m providerSetupRouteModel) handleIdleKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	if key.Type == tea.KeyRunes && string(key.Runes) == "s" {
		m.loading = true
		m.errText = ""
		return m, m.startSetupCmd()
	}
	if key.Type == tea.KeyRunes && string(key.Runes) == "f" {
		m.selected = providerSetupDefaultsFor(nextProviderFamily(m.selected.ProviderFamily))
		m.status = fmt.Sprintf("Selected provider family: %s (%s%s)", m.selected.ProviderFamily, m.selected.CanonicalHost, m.selected.CanonicalPathPrefix)
		return m, nil
	}
	if key.Type == tea.KeyRunes && string(key.Runes) == "r" {
		m.loading = true
		m.errText = ""
		m.status = "Refreshing broker-projected provider posture..."
		return m, m.refreshProfilesCmd()
	}
	return m, nil
}

func (m providerSetupRouteModel) startSetupCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		begin, err := m.client.ProviderSetupSessionBegin(ctx, brokerapi.ProviderSetupSessionBeginRequest{DisplayLabel: m.selected.DisplayLabel, ProviderFamily: m.selected.ProviderFamily, AdapterKind: m.selected.AdapterKind, CanonicalHost: m.selected.CanonicalHost, CanonicalPathPrefix: m.selected.CanonicalPathPrefix, AllowlistedModelIDs: append([]string{}, m.selected.AllowlistedModelIDs...)})
		if err != nil {
			return providerSetupRouteMsg{err: err}
		}
		prepare, err := m.client.ProviderSetupSecretIngressPrepare(ctx, brokerapi.ProviderSetupSecretIngressPrepareRequest{SetupSessionID: begin.SetupSession.SetupSessionID, IngressChannel: "tui_masked_input", CredentialField: "api_key"})
		if err != nil {
			return providerSetupRouteMsg{err: err}
		}
		return providerSetupRouteMsg{beginResp: &begin, prepareResp: &prepare}
	}
}

func (m providerSetupRouteModel) submitSecretCmd(secret []byte) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		submit, err := m.client.ProviderSetupSecretIngressSubmit(ctx, brokerapi.ProviderSetupSecretIngressSubmitRequest{SecretIngressToken: m.prepare.SecretIngressToken}, secret)
		if err != nil {
			return providerSetupRouteMsg{err: err}
		}
		return providerSetupRouteMsg{submitResp: &submit}
	}
}

func (m providerSetupRouteModel) refreshProfilesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		listResp, err := m.client.ProviderProfileList(ctx)
		if err != nil {
			return providerSetupRouteMsg{err: err}
		}
		return providerSetupRouteMsg{profileListResp: &listResp}
	}
}

func (m providerSetupRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return renderStateCard(routeLoadStateLoading, "Model Providers", "Applying broker-owned direct-credential setup flow...")
	}
	if m.errText != "" {
		return renderStateCard(routeLoadStateError, "Model Providers", m.errText)
	}
	masked := "(none)"
	if len(m.secretRunes) > 0 {
		masked = strings.Repeat("*", len(m.secretRunes))
	}
	current := m.currentProfile()
	supportedAuthModes := "(none)"
	if len(current.SupportedAuthModes) > 0 {
		supportedAuthModes = strings.Join(current.SupportedAuthModes, ", ")
	}
	return compactLines(
		sectionTitle("Model Providers")+" "+focusBadge(focus),
		"Broker-owned direct-credential setup session with trusted secret ingress.",
		"Raw secret values stay out of ordinary typed request/response bodies.",
		fmt.Sprintf("Selected provider family: %s endpoint=%s%s", valueOrNA(m.selected.ProviderFamily), valueOrNA(m.selected.CanonicalHost), valueOrNA(m.selected.CanonicalPathPrefix)),
		fmt.Sprintf("Current session: %s", valueOrNA(m.begin.SetupSession.SetupSessionID)),
		fmt.Sprintf("Current profile: %s", valueOrNA(current.ProviderProfileID)),
		fmt.Sprintf("Auth modes: supported=[%s] current=%s", supportedAuthModes, valueOrNA(current.CurrentAuthMode)),
		fmt.Sprintf("Compatibility posture: %s (readiness=%s)", valueOrNA(current.CompatibilityPosture), valueOrNA(current.ReadinessPosture.EffectiveReadiness)),
		fmt.Sprintf("Masked credential input: %s", masked),
		fmt.Sprintf("Broker-projected profiles discovered: %d", len(m.profiles)),
		m.status,
		keyHint("Route keys: s start setup, f switch family, r refresh posture; during entry type secret, Enter submit, Esc cancel"),
	)
}

func (m providerSetupRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	mainWidth := routeRegionWidth(ctx.Regions.Main, ctx.Width)
	mainHeight := routeRegionHeight(ctx.Regions.Main, ctx.Height)
	status := strings.TrimSpace(m.status)
	if status == "" && strings.TrimSpace(m.errText) != "" {
		status = m.errText
	}
	return routeSurface{
		Regions: routeSurfaceRegions{
			Main:   routeSurfaceRegion{Title: "Model providers", Body: m.View(mainWidth, mainHeight, ctx.Focus)},
			Bottom: routeSurfaceRegion{Body: keyHint("Route keys: s start setup, f switch family, r refresh posture; during entry type secret, Enter submit, Esc cancel")},
			Status: routeSurfaceRegion{Body: status},
		},
		Capabilities: routeSurfaceCapabilities{},
		Chrome:       routeSurfaceChrome{Breadcrumbs: []string{"Home", m.def.Label}},
	}
}

func (m providerSetupRouteModel) currentProfile() brokerapi.ProviderProfile {
	if strings.TrimSpace(m.submit.Profile.ProviderProfileID) != "" {
		return m.submit.Profile
	}
	if strings.TrimSpace(m.begin.Profile.ProviderProfileID) != "" {
		return m.begin.Profile
	}
	if len(m.profiles) > 0 {
		return m.profiles[0]
	}
	return brokerapi.ProviderProfile{}
}

func upsertProviderProfile(existing []brokerapi.ProviderProfile, profile brokerapi.ProviderProfile) []brokerapi.ProviderProfile {
	id := strings.TrimSpace(profile.ProviderProfileID)
	if id == "" {
		return existing
	}
	out := append([]brokerapi.ProviderProfile{}, existing...)
	for i := range out {
		if strings.TrimSpace(out[i].ProviderProfileID) == id {
			out[i] = profile
			return out
		}
	}
	return append(out, profile)
}

type providerSetupDefaults struct {
	DisplayLabel        string
	ProviderFamily      string
	AdapterKind         string
	CanonicalHost       string
	CanonicalPathPrefix string
	AllowlistedModelIDs []string
}

func providerSetupDefaultsFor(family string) providerSetupDefaults {
	switch strings.TrimSpace(family) {
	case "anthropic_compatible":
		return providerSetupDefaults{DisplayLabel: "Anthropic default", ProviderFamily: "anthropic_compatible", AdapterKind: "messages_v0", CanonicalHost: "api.anthropic.com", CanonicalPathPrefix: "/v1/messages", AllowlistedModelIDs: []string{"claude-3-5-sonnet-latest"}}
	default:
		return providerSetupDefaults{DisplayLabel: "OpenAI default", ProviderFamily: "openai_compatible", AdapterKind: "chat_completions_v0", CanonicalHost: "api.openai.com", CanonicalPathPrefix: "/v1/chat/completions", AllowlistedModelIDs: []string{"gpt-4.1-mini"}}
	}
}

func nextProviderFamily(current string) string {
	if strings.TrimSpace(current) == "anthropic_compatible" {
		return "openai_compatible"
	}
	return "anthropic_compatible"
}
