package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

var artifactClassFilters = []string{"", "diffs", "build_logs", "gate_evidence", "audit_verification_report", "approved_file_excerpts", "unapproved_file_excerpts"}

type artifactsLoadedMsg struct {
	items      []brokerapi.ArtifactSummary
	head       *brokerapi.LocalArtifactHeadResponse
	content    string
	contentErr string
	err        error
	seq        uint64
}

type artifactDetailMode string

const (
	artifactModeDiff   artifactDetailMode = "diff"
	artifactModeLog    artifactDetailMode = "log"
	artifactModeResult artifactDetailMode = "result"
)

type artifactsSelectDigestMsg struct {
	Digest string
}

type artifactsRouteModel struct {
	def          routeDefinition
	client       localBrokerClient
	loading      bool
	errText      string
	items        []brokerapi.ArtifactSummary
	selected     int
	classIndex   int
	active       *brokerapi.LocalArtifactHeadResponse
	content      string
	contentErr   string
	mode         artifactDetailMode
	presentation contentPresentationMode
	inspectorOn  bool
	loadSeq      uint64
	detailDoc    longFormDocumentState
}

func newArtifactsRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return artifactsRouteModel{def: def, client: client, inspectorOn: true, presentation: presentationRendered, detailDoc: newLongFormDocumentState()}
}

func (m artifactsRouteModel) ID() routeID { return m.def.ID }

func (m artifactsRouteModel) Title() string { return m.def.Label }

func (m artifactsRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		return m.handleRouteActivated(typed)
	case tea.KeyMsg:
		return m.handleKey(typed)
	case artifactsSelectDigestMsg:
		return m.handleSelectDigest(typed)
	case routeViewportScrollMsg:
		return m.handleViewportScroll(typed)
	case routeViewportResizeMsg:
		return m.handleViewportResize(typed)
	case artifactsLoadedMsg:
		return m.handleArtifactsLoaded(typed)
	default:
		return m, nil
	}
}

func (m artifactsRouteModel) handleRouteActivated(msg routeActivatedMsg) (routeModel, tea.Cmd) {
	if msg.RouteID != m.def.ID {
		return m, nil
	}
	return m.reload()
}

func (m artifactsRouteModel) handleSelectDigest(msg artifactsSelectDigestMsg) (routeModel, tea.Cmd) {
	digest := strings.TrimSpace(msg.Digest)
	if digest == "" {
		return m, nil
	}
	m.loading = true
	m.errText = ""
	m.loadSeq++
	return m, m.loadCmd(digest, m.loadSeq)
}

func (m artifactsRouteModel) handleViewportScroll(msg routeViewportScrollMsg) (routeModel, tea.Cmd) {
	if msg.Region == routeRegionInspector {
		m.detailDoc.Scroll(msg.Delta)
	}
	return m, nil
}

func (m artifactsRouteModel) handleViewportResize(msg routeViewportResizeMsg) (routeModel, tea.Cmd) {
	width, height := longFormViewportSizeForShell(msg.Width, msg.Height)
	m.detailDoc.Resize(width, height)
	return m, nil
}

func (m artifactsRouteModel) handleArtifactsLoaded(msg artifactsLoadedMsg) (routeModel, tea.Cmd) {
	if msg.seq != m.loadSeq {
		return m, nil
	}
	m.loading = false
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		return m, nil
	}
	m.errText = ""
	m.items = msg.items
	if m.selected >= len(m.items) {
		m.selected = 0
	}
	m.active = msg.head
	m.content = msg.content
	m.contentErr = msg.contentErr
	if m.active != nil {
		m.mode = preferredArtifactMode(m.active.Artifact.Reference.DataClass)
	}
	m.syncDetailDocument()
	return m, nil
}

func (m artifactsRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return renderStateCard(routeLoadStateLoading, "Artifacts", "Loading artifacts from typed broker artifact contracts...")
	}
	if m.errText != "" {
		return renderStateCard(routeLoadStateError, "Artifacts", "Load failed: "+m.errText+" (press r to retry)")
	}
	body := []string{
		sectionTitle("Artifacts") + " " + focusBadge(focus),
		fmt.Sprintf("Filter data_class=%q", artifactClassFilters[m.classIndex]),
		renderModeSwitchTabs([]string{string(presentationRendered), string(presentationRaw), string(presentationStructured)}, string(normalizePresentationMode(m.presentation))),
		renderDirectory("Artifact directory", renderArtifactDirectoryItems(m.items), m.selected),
	}
	body = append(body, keyHint("Route keys: j/k move, enter load detail, [/] class filter, m cycle detail mode, v cycle rendered/raw/structured, i toggle inspector, r reload"))
	return compactLines(body...)
}

func (m artifactsRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	breadcrumbs := []string{"Home", m.def.Label}
	if m.active != nil && strings.TrimSpace(m.active.Artifact.Reference.Digest) != "" {
		breadcrumbs = append(breadcrumbs, strings.TrimSpace(m.active.Artifact.Reference.Digest))
	}
	status := ""
	if strings.TrimSpace(m.errText) != "" {
		status = "Load failed: " + strings.TrimSpace(m.errText)
	} else if strings.TrimSpace(m.contentErr) != "" {
		status = "Content unavailable: " + strings.TrimSpace(m.contentErr)
	}
	inspector := ""
	if m.inspectorOn {
		inspector = renderArtifactInspector(m.active, m.mode, m.presentation, m.content, m.contentErr, &m.detailDoc)
	}
	return routeSurface{
		Regions: routeSurfaceRegions{
			Main:      routeSurfaceRegion{Title: "Artifact workspace", Body: m.View(ctx.Width, ctx.Height, ctx.Focus)},
			Inspector: routeSurfaceRegion{Title: "Artifact inspector", Body: inspector},
			Bottom:    routeSurfaceRegion{Body: keyHint("Route keys: j/k move, enter load detail, [/] class filter, m cycle detail mode, v cycle rendered/raw/structured, i toggle inspector, r reload")},
			Status:    routeSurfaceRegion{Body: status},
		},
		Chrome: routeSurfaceChrome{Breadcrumbs: breadcrumbs},
		Actions: routeSurfaceActions{
			ModeTabs:         []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
			ActiveTab:        string(normalizePresentationMode(m.presentation)),
			CopyActions:      artifactRouteCopyActions(m.active, m.content),
			ReferenceActions: artifactInspectorReferenceActions(m.active),
			LocalActions:     artifactInspectorLocalActions(),
		},
	}
}

func (m artifactsRouteModel) handleKey(key tea.KeyMsg) (routeModel, tea.Cmd) {
	s := key.String()
	if s == "r" {
		return m.reload()
	}
	if s == "i" {
		m.inspectorOn = !m.inspectorOn
		return m, nil
	}
	if s == "m" {
		m.mode = nextArtifactMode(m.mode)
		m.syncDetailDocument()
		return m, nil
	}
	if s == "v" {
		m.presentation = nextPresentationMode(m.presentation)
		m.syncDetailDocument()
		return m, nil
	}
	if s == "[" || s == "]" {
		m.rotateClassFilter(s == "]")
		return m.reload()
	}
	if s == "j" || s == "down" || s == "k" || s == "up" {
		m.moveSelection(s == "j" || s == "down")
		return m, nil
	}
	if s == "enter" {
		return m.loadSelectedArtifact()
	}
	return m, nil
}

func (m *artifactsRouteModel) rotateClassFilter(forward bool) {
	if forward {
		m.classIndex = (m.classIndex + 1) % len(artifactClassFilters)
		return
	}
	m.classIndex--
	if m.classIndex < 0 {
		m.classIndex = len(artifactClassFilters) - 1
	}
}

func (m *artifactsRouteModel) moveSelection(forward bool) {
	if len(m.items) == 0 {
		return
	}
	if forward {
		m.selected = (m.selected + 1) % len(m.items)
		return
	}
	m.selected--
	if m.selected < 0 {
		m.selected = len(m.items) - 1
	}
}

func (m artifactsRouteModel) loadSelectedArtifact() (routeModel, tea.Cmd) {
	if len(m.items) == 0 {
		return m, nil
	}
	m.loading = true
	m.errText = ""
	digest := m.items[m.selected].Reference.Digest
	m.loadSeq++
	return m, m.loadCmd(digest, m.loadSeq)
}

func (m artifactsRouteModel) reload() (routeModel, tea.Cmd) {
	m.loading = true
	m.errText = ""
	m.loadSeq++
	target := ""
	if m.selected >= 0 && m.selected < len(m.items) {
		target = m.items[m.selected].Reference.Digest
	}
	return m, m.loadCmd(target, m.loadSeq)
}

func (m artifactsRouteModel) loadCmd(digest string, seq uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		class := artifactClassFilters[m.classIndex]
		listResp, err := m.client.ArtifactList(ctx, 40, class)
		if err != nil {
			return artifactsLoadedMsg{err: err, seq: seq}
		}
		target := digest
		if target == "" && len(listResp.Artifacts) > 0 {
			target = listResp.Artifacts[0].Reference.Digest
		}
		if target == "" {
			return artifactsLoadedMsg{items: listResp.Artifacts, seq: seq}
		}
		headResp, err := m.client.ArtifactHead(ctx, target)
		if err != nil {
			return artifactsLoadedMsg{err: err, seq: seq}
		}
		readReq := brokerapi.ArtifactReadRequest{Digest: target, ProducerRole: "workspace", ConsumerRole: "model_gateway", DataClass: string(headResp.Artifact.Reference.DataClass)}
		events, readErr := m.client.ArtifactRead(ctx, readReq)
		if readErr != nil {
			return artifactsLoadedMsg{items: listResp.Artifacts, head: &headResp, contentErr: safeUIErrorText(readErr), seq: seq}
		}
		text, decodeErr := decodeArtifactStream(events)
		if decodeErr != nil {
			return artifactsLoadedMsg{items: listResp.Artifacts, head: &headResp, contentErr: safeUIErrorText(decodeErr), seq: seq}
		}
		return artifactsLoadedMsg{items: listResp.Artifacts, head: &headResp, content: text, seq: seq}
	}
}
