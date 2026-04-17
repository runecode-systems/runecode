package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type themePreset string

const (
	themePresetDark themePreset = "dark"
	themePresetDusk themePreset = "dusk"
	themePresetHigh themePreset = "high_contrast"
)

type themeTokens struct {
	SurfaceBase     lipgloss.Style
	SurfaceElevated lipgloss.Style
	SurfaceOverlay  lipgloss.Style
	BorderSubtle    lipgloss.Style
	BorderStrong    lipgloss.Style
	FocusRing       lipgloss.Style
	SelectionRing   lipgloss.Style
	TextPrimary     lipgloss.Style
	TextSecondary   lipgloss.Style
	TextTertiary    lipgloss.Style
	StateInfo       lipgloss.Style
	StateSuccess    lipgloss.Style
	StateWarn       lipgloss.Style
	StateDanger     lipgloss.Style
	AppTitle        lipgloss.Style
	SectionTitle    lipgloss.Style
	Muted           lipgloss.Style
	TableHeader     lipgloss.Style
	BadgeNeutral    lipgloss.Style
	BadgeInfo       lipgloss.Style
	BadgeSuccess    lipgloss.Style
	BadgeWarn       lipgloss.Style
	BadgeDanger     lipgloss.Style
	BadgeAdvisory   lipgloss.Style
	BadgeBlocking   lipgloss.Style
	BadgeReduced    lipgloss.Style
	BadgeProvDeg    lipgloss.Style
	BadgeAuditDeg   lipgloss.Style
	BadgeOverride   lipgloss.Style
	BadgeApproval   lipgloss.Style
	BadgeSystem     lipgloss.Style
	Selected        lipgloss.Style
	FocusLine       lipgloss.Style
	KeyHint         lipgloss.Style
	InspectorHint   lipgloss.Style
}

var appTheme = newTheme(themePresetDark)

func normalizeThemePreset(preset themePreset) themePreset {
	switch preset {
	case themePresetDark, themePresetDusk, themePresetHigh:
		return preset
	default:
		return themePresetDark
	}
}

func nextThemePreset(current themePreset) themePreset {
	switch normalizeThemePreset(current) {
	case themePresetDark:
		return themePresetDusk
	case themePresetDusk:
		return themePresetHigh
	default:
		return themePresetDark
	}
}

func newTheme(preset themePreset) themeTokens {
	switch normalizeThemePreset(preset) {
	case themePresetDusk:
		return newDuskTheme()
	case themePresetHigh:
		return newHighContrastTheme()
	case themePresetDark:
		return newDarkTheme()
	default:
		return newTheme(themePresetDark)
	}
}

func newDuskTheme() themeTokens {
	return themeTokens{
		SurfaceBase:     lipgloss.NewStyle().Foreground(lipgloss.Color("253")).Background(lipgloss.Color("235")),
		SurfaceElevated: lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("237")),
		SurfaceOverlay:  lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("60")),
		BorderSubtle:    lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
		BorderStrong:    lipgloss.NewStyle().Foreground(lipgloss.Color("182")).Bold(true),
		FocusRing:       lipgloss.NewStyle().Foreground(lipgloss.Color("183")).Bold(true),
		SelectionRing:   lipgloss.NewStyle().Foreground(lipgloss.Color("225")).Bold(true),
		TextPrimary:     lipgloss.NewStyle().Foreground(lipgloss.Color("255")),
		TextSecondary:   lipgloss.NewStyle().Foreground(lipgloss.Color("251")),
		TextTertiary:    lipgloss.NewStyle().Foreground(lipgloss.Color("247")),
		StateInfo:       lipgloss.NewStyle().Foreground(lipgloss.Color("195")).Background(lipgloss.Color("24")).Padding(0, 1),
		StateSuccess:    lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("29")).Padding(0, 1),
		StateWarn:       lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("215")).Padding(0, 1),
		StateDanger:     lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("161")).Padding(0, 1),
		AppTitle:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("183")),
		SectionTitle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("182")),
		Muted:           lipgloss.NewStyle().Foreground(lipgloss.Color("247")),
		TableHeader:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("224")),
		BadgeNeutral:    lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("240")).Padding(0, 1),
		BadgeInfo:       lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Padding(0, 1),
		BadgeSuccess:    lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("29")).Padding(0, 1),
		BadgeWarn:       lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("215")).Padding(0, 1),
		BadgeDanger:     lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("161")).Padding(0, 1),
		BadgeAdvisory:   lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("60")).Padding(0, 1),
		BadgeBlocking:   lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("178")).Padding(0, 1),
		BadgeReduced:    lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("96")).Padding(0, 1),
		BadgeProvDeg:    lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("173")).Padding(0, 1),
		BadgeAuditDeg:   lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("125")).Padding(0, 1),
		BadgeOverride:   lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("97")).Padding(0, 1),
		BadgeApproval:   lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("221")).Padding(0, 1),
		BadgeSystem:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("196")).Padding(0, 1),
		Selected:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("225")),
		FocusLine:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("183")),
		KeyHint:         lipgloss.NewStyle().Foreground(lipgloss.Color("186")),
		InspectorHint:   lipgloss.NewStyle().Foreground(lipgloss.Color("224")),
	}
}

func newHighContrastTheme() themeTokens {
	return themeTokens{
		SurfaceBase:     lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("16")),
		SurfaceElevated: lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("233")),
		SurfaceOverlay:  lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("229")),
		BorderSubtle:    lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Bold(true),
		BorderStrong:    lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true),
		FocusRing:       lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true).Underline(true),
		SelectionRing:   lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("18")).Bold(true),
		TextPrimary:     lipgloss.NewStyle().Foreground(lipgloss.Color("231")),
		TextSecondary:   lipgloss.NewStyle().Foreground(lipgloss.Color("254")),
		TextTertiary:    lipgloss.NewStyle().Foreground(lipgloss.Color("250")),
		StateInfo:       lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("51")).Padding(0, 1).Bold(true),
		StateSuccess:    lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("118")).Padding(0, 1).Bold(true),
		StateWarn:       lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("226")).Padding(0, 1).Bold(true),
		StateDanger:     lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("196")).Padding(0, 1).Bold(true),
		AppTitle:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")),
		SectionTitle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51")),
		Muted:           lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		TableHeader:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")),
		BadgeNeutral:    lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("250")).Padding(0, 1).Bold(true),
		BadgeInfo:       lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("51")).Padding(0, 1).Bold(true),
		BadgeSuccess:    lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("118")).Padding(0, 1).Bold(true),
		BadgeWarn:       lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("226")).Padding(0, 1).Bold(true),
		BadgeDanger:     lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("196")).Padding(0, 1).Bold(true),
		BadgeAdvisory:   lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("159")).Padding(0, 1).Bold(true),
		BadgeBlocking:   lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("220")).Padding(0, 1).Bold(true),
		BadgeReduced:    lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("129")).Padding(0, 1).Bold(true),
		BadgeProvDeg:    lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("214")).Padding(0, 1).Bold(true),
		BadgeAuditDeg:   lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("161")).Padding(0, 1).Bold(true),
		BadgeOverride:   lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("93")).Padding(0, 1).Bold(true),
		BadgeApproval:   lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("227")).Padding(0, 1).Bold(true),
		BadgeSystem:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("196")).Padding(0, 1),
		Selected:        lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("229")),
		FocusLine:       lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("51")),
		KeyHint:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51")),
		InspectorHint:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")),
	}
}

func newDarkTheme() themeTokens {
	return themeTokens{
		SurfaceBase:     lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("234")),
		SurfaceElevated: lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("236")),
		SurfaceOverlay:  lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("238")),
		BorderSubtle:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		BorderStrong:    lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true),
		FocusRing:       lipgloss.NewStyle().Foreground(lipgloss.Color("45")).Bold(true),
		SelectionRing:   lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true),
		TextPrimary:     lipgloss.NewStyle().Foreground(lipgloss.Color("255")),
		TextSecondary:   lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		TextTertiary:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		StateInfo:       lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("25")).Padding(0, 1),
		StateSuccess:    lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("28")).Padding(0, 1),
		StateWarn:       lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("220")).Padding(0, 1),
		StateDanger:     lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("160")).Padding(0, 1),
		AppTitle:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")),
		SectionTitle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")),
		Muted:           lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		TableHeader:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117")),
		BadgeNeutral:    lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("238")).Padding(0, 1),
		BadgeInfo:       lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("25")).Padding(0, 1),
		BadgeSuccess:    lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("28")).Padding(0, 1),
		BadgeWarn:       lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("220")).Padding(0, 1),
		BadgeDanger:     lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("160")).Padding(0, 1),
		BadgeAdvisory:   lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("63")).Padding(0, 1),
		BadgeBlocking:   lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("214")).Padding(0, 1),
		BadgeReduced:    lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("98")).Padding(0, 1),
		BadgeProvDeg:    lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("172")).Padding(0, 1),
		BadgeAuditDeg:   lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("125")).Padding(0, 1),
		BadgeOverride:   lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("93")).Padding(0, 1),
		BadgeApproval:   lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(lipgloss.Color("221")).Padding(0, 1),
		BadgeSystem:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("196")).Padding(0, 1),
		Selected:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51")),
		FocusLine:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("45")),
		KeyHint:         lipgloss.NewStyle().Foreground(lipgloss.Color("114")),
		InspectorHint:   lipgloss.NewStyle().Foreground(lipgloss.Color("222")),
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
