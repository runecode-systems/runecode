package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type inspectorContentKind string

const (
	inspectorContentTranscript inspectorContentKind = "transcript"
	inspectorContentDiff       inspectorContentKind = "diff"
	inspectorContentLog        inspectorContentKind = "log"
	inspectorContentMarkdown   inspectorContentKind = "markdown"
	inspectorContentRaw        inspectorContentKind = "raw"
	inspectorContentStructured inspectorContentKind = "structured"
)

type inspectorReference struct {
	Label string
	Items []string
}

type inspectorShellSpec struct {
	Title          string
	Summary        string
	Identity       string
	Status         string
	Badges         []string
	References     []inspectorReference
	LocalActions   []string
	ModeTabs       []string
	ActiveMode     string
	ContentKind    inspectorContentKind
	ContentLabel   string
	Content        string
	ViewportWidth  int
	ViewportHeight int
	CopyActions    []routeCopyAction
}

func renderDirectory(title string, items []string, selected int) string {
	if len(items) == 0 {
		return renderStateCard(routeLoadStateEmpty, title, "no items")
	}
	lines := make([]string, 0, len(items)+1)
	lines = append(lines, tableHeader(title))
	for i, item := range items {
		marker := " "
		if i == selected {
			marker = ">"
		}
		lines = append(lines, selectedLine(i == selected, fmt.Sprintf(" %s %s", marker, item)))
	}
	return compactLines(lines...)
}

func renderInspectorHeader(title string, badges ...string) string {
	line := strings.TrimSpace(title)
	if line == "" {
		line = "Inspector"
	} else if !strings.HasPrefix(strings.ToLower(line), "inspector") {
		line = "Inspector " + line
	}
	if len(badges) == 0 {
		return tableHeader(line)
	}
	return tableHeader(line) + " " + strings.Join(badges, " ")
}

func renderModeSwitchTabs(tabs []string, active string) string {
	if len(tabs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tabs))
	for _, tab := range tabs {
		if tab == active {
			parts = append(parts, "["+strings.ToUpper(tab)+"]")
			continue
		}
		parts = append(parts, tab)
	}
	return "Modes: " + strings.Join(parts, " | ")
}

func renderInspectorShell(spec inspectorShellSpec) string {
	title := strings.TrimSpace(spec.Title)
	if title == "" {
		title = "Inspector"
	}
	lines := []string{renderInspectorHeader(title, spec.Badges...)}
	lines = append(lines, renderInspectorOverview(spec)...)
	lines = append(lines, renderInspectorReferences(spec.References)...)
	lines = append(lines, renderInspectorActions(spec.LocalActions, spec.CopyActions)...)
	if tabs := renderModeSwitchTabs(spec.ModeTabs, spec.ActiveMode); strings.TrimSpace(tabs) != "" {
		lines = append(lines, tabs)
	}
	lines = append(lines, renderInspectorDetailViewport(spec)...)
	return compactLines(lines...)
}

func renderInspectorOverview(spec inspectorShellSpec) []string {
	lines := []string{tableHeader("Overview")}
	if summary := strings.TrimSpace(spec.Summary); summary != "" {
		lines = append(lines, "Summary: "+summary)
	}
	if identity := strings.TrimSpace(spec.Identity); identity != "" {
		lines = append(lines, "Identity: "+identity)
	}
	if status := strings.TrimSpace(spec.Status); status != "" {
		lines = append(lines, "Status: "+status)
	}
	if len(spec.ModeTabs) > 0 {
		lines = append(lines, fmt.Sprintf("Summary → detail: %s mode", strings.ToUpper(strings.TrimSpace(spec.ActiveMode))))
	}
	return lines
}

func renderInspectorReferences(refs []inspectorReference) []string {
	if len(refs) == 0 {
		return nil
	}
	lines := []string{tableHeader("Linked references")}
	for _, ref := range refs {
		lines = append(lines, renderLinkedReferenceLine("Linked "+defaultInspectorReferenceLabel(ref.Label), ref.Items))
	}
	return lines
}

func defaultInspectorReferenceLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "linked"
	}
	return label
}

func renderInspectorActions(local []string, copyActions []routeCopyAction) []string {
	if len(local) == 0 && len(copyActions) == 0 {
		return nil
	}
	lines := []string{tableHeader("Actions")}
	if len(local) > 0 {
		lines = append(lines, "Local actions: "+strings.Join(local, " | "))
	}
	if labels := inspectorCopyActionLabels(copyActions); len(labels) > 0 {
		lines = append(lines, "Copy actions: "+strings.Join(labels, " | "))
	}
	return lines
}

func inspectorCopyActionLabels(copyActions []routeCopyAction) []string {
	labels := make([]string, 0, len(copyActions))
	for _, action := range copyActions {
		label := strings.TrimSpace(action.Label)
		if label == "" {
			label = strings.TrimSpace(action.ID)
		}
		if label == "" {
			continue
		}
		labels = append(labels, label)
	}
	return labels
}

func renderInspectorDetailViewport(spec inspectorShellSpec) []string {
	contentLabel := strings.TrimSpace(spec.ContentLabel)
	if contentLabel == "" {
		contentLabel = "content"
	}
	renderedContent := renderInspectorLongForm(spec.ContentKind, spec.Content, spec.ViewportWidth, spec.ViewportHeight)
	return []string{tableHeader("Detail viewport"), "Long-form " + contentLabel + ":", renderedContent}
}

func renderStateCard(state routeLoadState, title, message string) string {
	label := string(state)
	if label == "" {
		label = string(routeLoadStateReady)
	}
	if strings.TrimSpace(message) == "" {
		message = "n/a"
	}
	return compactLines(
		tableHeader(strings.ToUpper(label))+" "+title,
		message,
	)
}

type longFormViewport struct {
	model viewport.Model
	ready bool
}

func (v *longFormViewport) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}
	if !v.ready {
		v.model = viewport.New(width, height)
		v.ready = true
		return
	}
	v.model.Width = width
	v.model.Height = height
}

func (v *longFormViewport) SetContent(text string) {
	if !v.ready {
		v.SetSize(80, 20)
	}
	v.model.SetContent(text)
}

func (v *longFormViewport) View() string {
	if !v.ready {
		v.SetSize(80, 20)
	}
	return v.model.View()
}

func renderLongFormViewport(text string, width, height int) string {
	var vp longFormViewport
	vp.SetSize(width, height)
	vp.SetContent(text)
	return vp.View()
}

func normalizeLongFormViewportSize(width, height int) (int, int) {
	if width <= 0 {
		width = 96
	}
	if width > 120 {
		width = 120
	}
	if height <= 0 {
		height = 12
	}
	if height > 24 {
		height = 24
	}
	return width, height
}

func renderInspectorLongForm(kind inspectorContentKind, text string, width, height int) string {
	if strings.TrimSpace(text) == "" {
		text = "(empty)"
	}
	width, height = normalizeLongFormViewportSize(width, height)
	label := strings.TrimSpace(string(kind))
	if label == "" {
		label = "content"
	}
	body := renderLongFormViewport(formatInspectorLongForm(kind, text), width, height)
	return compactLines(fmt.Sprintf("[%s viewport %dx%d]", label, width, height), body)
}

func formatInspectorLongForm(kind inspectorContentKind, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "(empty)"
	}
	switch kind {
	case inspectorContentDiff:
		return formatDiffContent(text)
	case inspectorContentMarkdown:
		return formatMarkdownContent(text)
	case inspectorContentStructured:
		return formatStructuredContent(text)
	default:
		return text
	}
}

func formatDiffContent(text string) string {
	lines := strings.Split(text, "\n")
	add, del := 0, 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "+") && !strings.HasPrefix(trimmed, "+++") {
			add++
		}
		if strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "---") {
			del++
		}
	}
	head := fmt.Sprintf("Diff summary: lines=%d additions=%d deletions=%d", len(lines), add, del)
	return head + "\n" + strings.Join(lines, "\n")
}

func formatMarkdownContent(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			lines[i] = "§ " + strings.TrimLeft(trimmed, "# ")
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			lines[i] = "• " + strings.TrimSpace(trimmed[2:])
		}
	}
	return "Markdown reading view:\n" + strings.Join(lines, "\n")
}

func formatStructuredContent(text string) string {
	lines := strings.Split(text, "\n")
	structuredLines := 0
	totalNonEmpty := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		totalNonEmpty++
		if isStructuredKVLine(trimmed) {
			structuredLines++
		}
	}
	if totalNonEmpty == 0 || structuredLines*2 < totalNonEmpty {
		return text
	}
	formatted := make([]string, 0, len(lines)+1)
	formatted = append(formatted, "Structured reading view:")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isStructuredKVLine(trimmed) {
			parts := strings.SplitN(trimmed, "=", 2)
			formatted = append(formatted, fmt.Sprintf("%s) %s = %s", strconv.Itoa(i+1), strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])))
			continue
		}
		formatted = append(formatted, fmt.Sprintf("%s) %s", strconv.Itoa(i+1), trimmed))
	}
	if len(formatted) == 1 {
		formatted = append(formatted, "(no fields)")
	}
	return strings.Join(formatted, "\n")
}

func isStructuredKVLine(line string) bool {
	if !strings.Contains(line, "=") {
		return false
	}
	parts := strings.SplitN(line, "=", 2)
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return false
	}
	if strings.Contains(key, " ") || strings.Contains(key, ":") {
		return false
	}
	return true
}

type composeTextarea struct {
	model textarea.Model
	set   bool
}

func newComposeTextarea() composeTextarea {
	t := textarea.New()
	t.Placeholder = "Type a message"
	t.CharLimit = 4000
	t.Prompt = "┃ "
	t.SetHeight(3)
	return composeTextarea{model: t, set: true}
}

func (c *composeTextarea) ensure() {
	if c.set {
		return
	}
	*c = newComposeTextarea()
}

func (c *composeTextarea) Value() string {
	c.ensure()
	return c.model.Value()
}

func (c *composeTextarea) SetValue(value string) {
	c.ensure()
	c.model.SetValue(value)
}

func (c *composeTextarea) Focus() {
	c.ensure()
	c.model.Focus()
}

func (c *composeTextarea) Blur() {
	c.ensure()
	c.model.Blur()
}

func (c *composeTextarea) BubbleUpdate(msg tea.Msg) {
	c.ensure()
	c.model, _ = c.model.Update(msg)
}

func (c *composeTextarea) View() string {
	c.ensure()
	return c.model.View()
}
