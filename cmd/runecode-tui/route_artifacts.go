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
}

func newArtifactsRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return artifactsRouteModel{def: def, client: client, inspectorOn: true, presentation: presentationRendered}
}

func (m artifactsRouteModel) ID() routeID { return m.def.ID }

func (m artifactsRouteModel) Title() string { return m.def.Label }

func (m artifactsRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		if typed.RouteID != m.def.ID {
			return m, nil
		}
		return m.reload()
	case tea.KeyMsg:
		return m.handleKey(typed)
	case artifactsSelectDigestMsg:
		digest := strings.TrimSpace(typed.Digest)
		if digest == "" {
			return m, nil
		}
		m.loading = true
		m.errText = ""
		m.loadSeq++
		return m, m.loadCmd(digest, m.loadSeq)
	case artifactsLoadedMsg:
		if typed.seq != m.loadSeq {
			return m, nil
		}
		m.loading = false
		if typed.err != nil {
			m.errText = safeUIErrorText(typed.err)
			return m, nil
		}
		m.errText = ""
		m.items = typed.items
		if m.selected >= len(m.items) {
			m.selected = 0
		}
		m.active = typed.head
		m.content = typed.content
		m.contentErr = typed.contentErr
		if m.active != nil {
			m.mode = preferredArtifactMode(m.active.Artifact.Reference.DataClass)
		}
		return m, nil
	default:
		return m, nil
	}
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
	if m.inspectorOn {
		body = append(body, renderArtifactInspector(m.active, m.mode, m.presentation, m.content, m.contentErr))
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
	return routeSurface{
		Main:           m.View(ctx.Width, ctx.Height, ctx.Focus),
		Inspector:      renderArtifactInspector(m.active, m.mode, m.presentation, m.content, m.contentErr),
		BottomStrip:    keyHint("Route keys: j/k move, enter load detail, [/] class filter, m cycle detail mode, v cycle rendered/raw/structured, i toggle inspector, r reload"),
		Status:         status,
		Breadcrumbs:    breadcrumbs,
		MainTitle:      "Artifact workspace",
		InspectorTitle: "Artifact inspector",
		ModeTabs:       []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveTab:      string(normalizePresentationMode(m.presentation)),
		CopyActions:    artifactRouteCopyActions(m.active, m.content),
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
		return m, nil
	}
	if s == "v" {
		m.presentation = nextPresentationMode(m.presentation)
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

func renderArtifactList(items []brokerapi.ArtifactSummary, selected int) string {
	if len(items) == 0 {
		return "  - no artifacts"
	}
	line := ""
	for i, item := range items {
		marker := " "
		if i == selected {
			marker = ">"
		}
		line += selectedLine(i == selected, fmt.Sprintf("  %s %s class=%s bytes=%d run=%s", marker, item.Reference.Digest, item.Reference.DataClass, item.Reference.SizeBytes, item.RunID)) + "\n"
	}
	return line
}

func renderArtifactDirectoryItems(items []brokerapi.ArtifactSummary) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprintf("%s class=%s bytes=%d", item.Reference.Digest, item.Reference.DataClass, item.Reference.SizeBytes))
	}
	return out
}

func renderArtifactInspector(head *brokerapi.LocalArtifactHeadResponse, mode artifactDetailMode, presentation contentPresentationMode, content, contentErr string) string {
	if head == nil {
		return "  Select an artifact and press enter to load detail."
	}
	a := head.Artifact
	mode = normalizeArtifactMode(mode)
	presentation = normalizePresentationMode(presentation)
	contentView := renderArtifactContent(mode, presentation, content, contentErr)
	kind := artifactContentKind(mode, a.Reference.ContentType, presentation)
	return renderInspectorShell(inspectorShellSpec{
		Title:        "Artifact inspector",
		Summary:      fmt.Sprintf("artifact=%s class=%s bytes=%d", a.Reference.Digest, a.Reference.DataClass, a.Reference.SizeBytes),
		Identity:     fmt.Sprintf("digest=%s", a.Reference.Digest),
		Status:       fmt.Sprintf("data_class=%s content_type=%s", a.Reference.DataClass, a.Reference.ContentType),
		Badges:       []string{stateBadgeWithLabel("class", fmt.Sprintf("%v", a.Reference.DataClass)), appTheme.InspectorHint.Render("typed metadata first")},
		References:   []inspectorReference{{Label: "run", Items: []string{a.RunID}}},
		LocalActions: []string{"jump:runs", "jump:audit", "copy:digest", "copy:provenance_receipt"},
		CopyActions:  artifactRouteCopyActions(head, content),
		ModeTabs:     []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveMode:   string(presentation),
		ContentKind:  kind,
		ContentLabel: fmt.Sprintf("%s content", mode),
		Content: compactLines(
			fmt.Sprintf("Data class: %s", a.Reference.DataClass),
			fmt.Sprintf("Typed detail mode: %s (metadata remains control-plane truth)", mode),
			fmt.Sprintf("Presentation mode: %s", presentation),
			fmt.Sprintf("Provenance receipt: %s", a.Reference.ProvenanceReceiptHash),
			"Inspectable content is supplemental evidence, not authoritative run/approval truth.",
			contentView,
		),
		ViewportWidth:  96,
		ViewportHeight: 14,
	})
}

func artifactRouteCopyActions(head *brokerapi.LocalArtifactHeadResponse, content string) []routeCopyAction {
	if head == nil {
		return nil
	}
	ref := head.Artifact.Reference
	preview := strings.TrimSpace(content)
	if preview != "" {
		lines := strings.Split(preview, "\n")
		if len(lines) > 8 {
			preview = strings.Join(lines[:8], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-8)
		}
	}
	return compactCopyActions([]routeCopyAction{
		{ID: "digest", Label: "artifact digest", Text: ref.Digest},
		{ID: "provenance_receipt", Label: "provenance receipt", Text: ref.ProvenanceReceiptHash},
		{ID: "artifact_preview", Label: "artifact preview", Text: preview},
	})
}

func artifactContentKind(mode artifactDetailMode, contentType string, presentation contentPresentationMode) inspectorContentKind {
	if presentation == presentationRaw {
		return inspectorContentRaw
	}
	if presentation == presentationStructured {
		return inspectorContentStructured
	}
	lowerType := strings.ToLower(strings.TrimSpace(contentType))
	if strings.Contains(lowerType, "markdown") {
		return inspectorContentMarkdown
	}
	switch mode {
	case artifactModeDiff:
		return inspectorContentDiff
	case artifactModeLog:
		return inspectorContentLog
	default:
		return inspectorContentRaw
	}
}

func preferredArtifactMode(dataClass any) artifactDetailMode {
	value := strings.ToLower(fmt.Sprintf("%v", dataClass))
	switch {
	case strings.Contains(value, "diff"):
		return artifactModeDiff
	case strings.Contains(value, "log"):
		return artifactModeLog
	default:
		return artifactModeResult
	}
}

func normalizeArtifactMode(mode artifactDetailMode) artifactDetailMode {
	switch mode {
	case artifactModeDiff, artifactModeLog, artifactModeResult:
		return mode
	default:
		return artifactModeResult
	}
}

func nextArtifactMode(current artifactDetailMode) artifactDetailMode {
	switch normalizeArtifactMode(current) {
	case artifactModeDiff:
		return artifactModeLog
	case artifactModeLog:
		return artifactModeResult
	default:
		return artifactModeDiff
	}
}

func renderArtifactContent(mode artifactDetailMode, presentation contentPresentationMode, content, contentErr string) string {
	presentation = normalizePresentationMode(presentation)
	if contentErr != "" {
		return fmt.Sprintf("  %s content unavailable: %s", mode, contentErr)
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Sprintf("  %s content unavailable for current artifact.", mode)
	}
	if presentation == presentationRaw {
		return fmt.Sprintf("  %s raw (secrets redacted):\n%s", mode, redactSecrets(content))
	}
	lines := strings.Split(content, "\n")
	if presentation == presentationStructured {
		first := strings.TrimSpace(lines[0])
		last := strings.TrimSpace(lines[len(lines)-1])
		first = redactSecrets(first)
		last = redactSecrets(last)
		if len(lines) > 12 {
			return fmt.Sprintf("  %s structured:\n  - lines=%d\n  - non_empty=%d\n  - preview_first=%q\n  - preview_last=%q", mode, len(lines), countNonEmptyLines(lines), first, last)
		}
		return fmt.Sprintf("  %s structured:\n  - lines=%d\n  - non_empty=%d\n  - preview=%q", mode, len(lines), countNonEmptyLines(lines), redactSecrets(strings.TrimSpace(content)))
	}
	if len(lines) > 10 {
		return fmt.Sprintf("  %s preview (secrets redacted):\n%s\n  ... (%d more lines)", mode, redactSecrets(strings.Join(lines[:10], "\n")), len(lines)-10)
	}
	return fmt.Sprintf("  %s preview (secrets redacted):\n%s", mode, redactSecrets(content))
}

func countNonEmptyLines(lines []string) int {
	total := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			total++
		}
	}
	return total
}
