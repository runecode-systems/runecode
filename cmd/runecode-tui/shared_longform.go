package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
)

type longFormViewport struct {
	model viewport.Model
	ready bool
}

type longFormViewportState struct {
	Width   int
	Height  int
	YOffset int
}

type longFormDocumentState struct {
	ObjectRef         workbenchObjectRef
	Kind              inspectorContentKind
	Label             string
	RawContent        string
	FormattedContent  string
	Viewport          longFormViewportState
	lastDocumentToken string
}

func newLongFormDocumentState() longFormDocumentState {
	width, height := normalizeLongFormViewportSize(0, 0)
	return longFormDocumentState{Viewport: longFormViewportState{Width: width, Height: height}}
}

func (d *longFormDocumentState) SetDocument(ref workbenchObjectRef, kind inspectorContentKind, label, content string) {
	if d.Viewport.Width <= 0 || d.Viewport.Height <= 0 {
		d.Viewport.Width, d.Viewport.Height = normalizeLongFormViewportSize(0, 0)
	}
	documentToken := longFormDocumentToken(ref, kind)
	if d.lastDocumentToken != documentToken {
		d.Viewport.YOffset = 0
	}
	d.ObjectRef = ref
	d.Kind = kind
	d.Label = strings.TrimSpace(label)
	d.RawContent = content
	d.FormattedContent = formatInspectorLongForm(kind, content)
	d.lastDocumentToken = documentToken
	d.clampOffset()
}

func (d *longFormDocumentState) Resize(width, height int) {
	d.Viewport.Width, d.Viewport.Height = normalizeLongFormViewportSize(width, height)
	d.clampOffset()
}

func (d *longFormDocumentState) Scroll(delta int) {
	if delta == 0 {
		return
	}
	d.Viewport.YOffset += delta
	d.clampOffset()
}

func (d *longFormDocumentState) clampOffset() {
	if d.Viewport.YOffset < 0 {
		d.Viewport.YOffset = 0
	}
	max := d.maxYOffset()
	if d.Viewport.YOffset > max {
		d.Viewport.YOffset = max
	}
}

func (d longFormDocumentState) maxYOffset() int {
	lines := 1
	text := strings.TrimSpace(d.FormattedContent)
	if text != "" {
		lines = strings.Count(d.FormattedContent, "\n") + 1
	}
	max := lines - d.Viewport.Height
	if max < 0 {
		return 0
	}
	return max
}

func (d longFormDocumentState) contentLabel() string {
	label := strings.TrimSpace(d.Label)
	if label == "" {
		return "content"
	}
	return label
}

func (d longFormDocumentState) Render() string {
	text := d.FormattedContent
	if strings.TrimSpace(text) == "" {
		text = "(empty)"
	}
	label := strings.TrimSpace(string(d.Kind))
	if label == "" {
		label = "content"
	}
	width, height := normalizeLongFormViewportSize(d.Viewport.Width, d.Viewport.Height)
	vp := viewport.New(width, height)
	vp.SetContent(text)
	yOffset := d.Viewport.YOffset
	if yOffset < 0 {
		yOffset = 0
	}
	vp.YOffset = yOffset
	return compactLines(fmt.Sprintf("[%s viewport %dx%d offset=%d]", label, width, height, yOffset), vp.View())
}

func longFormDocumentToken(ref workbenchObjectRef, kind inspectorContentKind) string {
	parts := []string{strings.TrimSpace(ref.Kind), strings.TrimSpace(ref.ID), strings.TrimSpace(ref.WorkspaceID), strings.TrimSpace(ref.SessionID), strings.TrimSpace(string(kind))}
	return strings.Join(parts, "|")
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
		height = 14
	}
	if height > 24 {
		height = 24
	}
	return width, height
}

func longFormViewportSizeForShell(width, height int) (int, int) {
	if width <= 0 {
		width = 120
	}
	if height <= 0 {
		height = 40
	}
	vw := width/3 - 8
	vh := height - 24
	if vw < 48 {
		vw = width - 16
	}
	if vh < 8 {
		vh = 8
	}
	return normalizeLongFormViewportSize(vw, vh)
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
