package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type themePreset string

const (
	themePresetDark themePreset = "dark"
)

type themeTokens struct {
	AppTitle      lipgloss.Style
	SectionTitle  lipgloss.Style
	Muted         lipgloss.Style
	TableHeader   lipgloss.Style
	BadgeNeutral  lipgloss.Style
	BadgeInfo     lipgloss.Style
	BadgeSuccess  lipgloss.Style
	BadgeWarn     lipgloss.Style
	BadgeDanger   lipgloss.Style
	BadgeAdvisory lipgloss.Style
	BadgeBlocking lipgloss.Style
	BadgeReduced  lipgloss.Style
	BadgeProvDeg  lipgloss.Style
	BadgeAuditDeg lipgloss.Style
	BadgeOverride lipgloss.Style
	BadgeApproval lipgloss.Style
	BadgeSystem   lipgloss.Style
	Selected      lipgloss.Style
	FocusLine     lipgloss.Style
	KeyHint       lipgloss.Style
	InspectorHint lipgloss.Style
}

var appTheme = newTheme(themePresetDark)

func newTheme(preset themePreset) themeTokens {
	switch preset {
	case themePresetDark:
		return themeTokens{
			AppTitle:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")),
			SectionTitle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")),
			Muted:         lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
			TableHeader:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117")),
			BadgeNeutral:  lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("238")).Padding(0, 1),
			BadgeInfo:     lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("25")).Padding(0, 1),
			BadgeSuccess:  lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("28")).Padding(0, 1),
			BadgeWarn:     lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("220")).Padding(0, 1),
			BadgeDanger:   lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("160")).Padding(0, 1),
			BadgeAdvisory: lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("63")).Padding(0, 1),
			BadgeBlocking: lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("214")).Padding(0, 1),
			BadgeReduced:  lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("98")).Padding(0, 1),
			BadgeProvDeg:  lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("172")).Padding(0, 1),
			BadgeAuditDeg: lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("125")).Padding(0, 1),
			BadgeOverride: lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("93")).Padding(0, 1),
			BadgeApproval: lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("221")).Padding(0, 1),
			BadgeSystem:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("196")).Padding(0, 1),
			Selected:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51")),
			FocusLine:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("45")),
			KeyHint:       lipgloss.NewStyle().Foreground(lipgloss.Color("114")),
			InspectorHint: lipgloss.NewStyle().Foreground(lipgloss.Color("222")),
		}
	default:
		return newTheme(themePresetDark)
	}
}

func sectionTitle(label string) string {
	return appTheme.SectionTitle.Render(label)
}

func tableHeader(label string) string {
	return appTheme.TableHeader.Render(label)
}

func muted(label string) string {
	return appTheme.Muted.Render(label)
}

func keyHint(label string) string {
	return appTheme.KeyHint.Render(label)
}

func selectedLine(selected bool, line string) string {
	if !selected {
		return line
	}
	return appTheme.Selected.Render(line)
}

func focusBadge(focus focusArea) string {
	return infoBadge(fmt.Sprintf("focus=%s", strings.ToUpper(focus.Label())))
}

func navStateBadge(active bool) string {
	if active {
		return successBadge("ACTIVE")
	}
	return neutralBadge("IDLE")
}

func boolBadge(label string, value bool) string {
	if value {
		return successBadge(label + "=true")
	}
	return warnBadge(label + "=false")
}

func stateBadgeWithLabel(label, status string) string {
	return label + "=" + postureBadge(status)
}

func postureBadge(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case "ok", "ready", "approved", "consumed", "active", "nominal", "healthy", "anchored", "sandboxed":
		return successBadge(strings.ToUpper(normalized))
	case "degraded", "warning", "stale", "superseded", "expired", "pending", "wait", "unanchored", "reduced", "advisory":
		return warnBadge(strings.ToUpper(normalized))
	case "failed", "invalid", "denied", "error", "blocked", "unhealthy":
		return dangerBadge(strings.ToUpper(normalized))
	default:
		if normalized == "" {
			return neutralBadge("N/A")
		}
		return infoBadge(strings.ToUpper(normalized))
	}
}

func neutralBadge(label string) string {
	return appTheme.BadgeNeutral.Render("• " + label)
}

func infoBadge(label string) string {
	return appTheme.BadgeInfo.Render("i " + label)
}

func successBadge(label string) string {
	return appTheme.BadgeSuccess.Render("✓ " + label)
}

func warnBadge(label string) string {
	return appTheme.BadgeWarn.Render("! " + label)
}

func dangerBadge(label string) string {
	return appTheme.BadgeDanger.Render("✕ " + label)
}

func advisoryBadge(label string) string {
	return appTheme.BadgeAdvisory.Render("A " + label)
}

func blockingBadge(label string) string {
	return appTheme.BadgeBlocking.Render("B " + label)
}

func reducedAssuranceBadge(label string) string {
	return appTheme.BadgeReduced.Render("R " + label)
}

func provisioningDegradedBadge(label string) string {
	return appTheme.BadgeProvDeg.Render("P " + label)
}

func auditDegradedBadge(label string) string {
	return appTheme.BadgeAuditDeg.Render("U " + label)
}

func gateOverrideBadge(label string) string {
	return appTheme.BadgeOverride.Render("O " + label)
}

func approvalRequiredBadge(label string) string {
	return appTheme.BadgeApproval.Render("? " + label)
}

func systemFailureBadge(label string) string {
	return appTheme.BadgeSystem.Render("S " + label)
}
