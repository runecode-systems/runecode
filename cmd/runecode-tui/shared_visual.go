package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type visualTone string

const (
	visualToneNeutral   visualTone = "neutral"
	visualToneInfo      visualTone = "info"
	visualToneSuccess   visualTone = "success"
	visualToneAttention visualTone = "attention"
	visualToneDanger    visualTone = "danger"
	visualToneCommand   visualTone = "command"
)

type productCardSpec struct {
	Tone  visualTone
	Title string
	Badge string
	Lines []string
}

func renderProductCard(spec productCardSpec) string {
	title := strings.TrimSpace(spec.Title)
	if title == "" {
		title = "Status"
	}
	header := tableHeader(title)
	if badge := strings.TrimSpace(spec.Badge); badge != "" {
		header += " " + badge
	}
	lines := []string{header}
	for _, line := range spec.Lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return renderAccentRail(spec.Tone, compactLines(lines...))
}

func renderAccentRail(tone visualTone, body string) string {
	body = strings.TrimRight(body, "\n")
	if strings.TrimSpace(body) == "" {
		return ""
	}
	rail := toneAccentStyle(tone).Render("|")
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		lines[i] = rail + " " + line
	}
	return strings.Join(lines, "\n")
}

func toneAccentStyle(tone visualTone) lipgloss.Style {
	switch tone {
	case visualToneSuccess:
		return appTheme.StateSuccess.Bold(true)
	case visualToneAttention:
		return appTheme.StateWarn.Bold(true)
	case visualToneDanger:
		return appTheme.StateDanger.Bold(true)
	case visualToneCommand:
		return appTheme.Selected.Bold(true)
	case visualToneInfo:
		return appTheme.StateInfo.Bold(true)
	default:
		return appTheme.BorderStrong.Bold(true)
	}
}

func toneBadge(tone visualTone, label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		label = "STATUS"
	}
	switch tone {
	case visualToneSuccess:
		return successBadge(label)
	case visualToneAttention:
		return warnBadge(label)
	case visualToneDanger:
		return dangerBadge(label)
	case visualToneCommand:
		return infoBadge(label)
	case visualToneInfo:
		return infoBadge(label)
	default:
		return neutralBadge(label)
	}
}

func clipDisplayText(text string, width int) string {
	if width <= 0 || lipgloss.Width(text) <= width {
		return text
	}
	if width <= 3 {
		return clipDisplayRunes(text, width)
	}
	limit := width - 3
	return clipDisplayRunes(text, limit) + "..."
}

func clipDisplayRunes(text string, width int) string {
	if width <= 0 {
		return ""
	}
	var b strings.Builder
	for _, r := range text {
		candidate := b.String() + string(r)
		if lipgloss.Width(candidate) > width {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

func rightAlignMetadata(left, right string, width int) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if right == "" || width <= 0 {
		return left
	}
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	if leftWidth+rightWidth+2 > width {
		left = clipDisplayText(left, width-rightWidth-2)
		leftWidth = lipgloss.Width(left)
	}
	gap := width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + muted(right)
}
