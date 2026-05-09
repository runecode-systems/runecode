//go:build runecode_tui_snapshot

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/muesli/termenv"
)

var snapshotRenderMu sync.Mutex

type snapshotManifest struct {
	Version   int                     `json:"version"`
	Width     int                     `json:"width"`
	Height    int                     `json:"height"`
	Viewport  string                  `json:"viewport,omitempty"`
	Theme     string                  `json:"theme"`
	Scenarios []snapshotManifestEntry `json:"scenarios"`
}

type snapshotManifestArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256,omitempty"`
}

type snapshotManifestArtifacts struct {
	ANSI snapshotManifestArtifact `json:"ansi"`
	Text snapshotManifestArtifact `json:"text"`
	SVG  snapshotManifestArtifact `json:"svg"`
	PNG  snapshotManifestArtifact `json:"png"`
}

type snapshotManifestEntry struct {
	Name      string                    `json:"name"`
	Viewport  string                    `json:"viewport,omitempty"`
	Width     int                       `json:"width"`
	Height    int                       `json:"height"`
	Route     string                    `json:"route"`
	Artifacts snapshotManifestArtifacts `json:"artifacts"`
}

type snapshotArtifactPaths struct {
	ANSI string
	Text string
	SVG  string
	PNG  string
}

type snapshotSVGLayout struct {
	Cols       int
	Rows       int
	CellWidth  int
	CellHeight int
	FontSize   int
	Padding    int
	Width      int
	Height     int
}

func writeSnapshotArtifacts(cfg tuiSnapshotConfig) error {
	cfg, err := normalizeSnapshotConfig(cfg)
	if err != nil {
		return err
	}
	scenarios, err := resolveSnapshotScenarios(cfg.scenario)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.outputDir, 0o755); err != nil {
		return err
	}
	manifest := snapshotManifest{Version: 2, Width: cfg.width, Height: cfg.height, Viewport: string(cfg.viewport), Theme: string(snapshotManifestTheme(cfg, scenarios))}
	entries, err := writeSnapshotScenarioArtifacts(cfg, scenarios)
	if err != nil {
		return err
	}
	manifest.Scenarios = entries
	return writeSnapshotManifest(cfg.outputDir, manifest)
}

func normalizeSnapshotConfig(cfg tuiSnapshotConfig) (tuiSnapshotConfig, error) {
	if cfg.outputDir == "" {
		cfg.outputDir = snapshotDefaultOutputDir()
	}
	normalizedOutputDir, err := normalizeSnapshotOutputDir(cfg.outputDir)
	if err != nil {
		return tuiSnapshotConfig{}, err
	}
	cfg.outputDir = normalizedOutputDir
	viewport, err := resolveSnapshotViewportPreset(cfg.viewport)
	if err != nil {
		return tuiSnapshotConfig{}, err
	}
	if cfg.viewport == "" {
		cfg.viewport = viewport.Name
	}
	if cfg.width == 0 {
		cfg.width = viewport.Width
	}
	if cfg.height == 0 {
		cfg.height = viewport.Height
	}
	if cfg.width <= 0 || cfg.height <= 0 {
		return tuiSnapshotConfig{}, fmt.Errorf("snapshot dimensions must be positive")
	}
	return cfg, nil
}

func snapshotManifestTheme(cfg tuiSnapshotConfig, scenarios []snapshotScenarioState) themePreset {
	if cfg.theme != "" {
		return cfg.theme
	}
	if len(scenarios) > 0 {
		return scenarios[0].Theme
	}
	return themePresetDark
}

func writeSnapshotScenarioArtifacts(cfg tuiSnapshotConfig, scenarios []snapshotScenarioState) ([]snapshotManifestEntry, error) {
	entries := make([]snapshotManifestEntry, 0, len(scenarios))
	for _, scenario := range scenarios {
		entry, err := writeSnapshotScenarioArtifact(cfg, scenario)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func writeSnapshotScenarioArtifact(cfg tuiSnapshotConfig, scenario snapshotScenarioState) (snapshotManifestEntry, error) {
	paths := snapshotPathsForScenario(cfg.outputDir, scenario.Name, cfg.viewport)
	ansiManifestPath, err := snapshotManifestArtifactPath(cfg.outputDir, paths.ANSI)
	if err != nil {
		return snapshotManifestEntry{}, err
	}
	textManifestPath, err := snapshotManifestArtifactPath(cfg.outputDir, paths.Text)
	if err != nil {
		return snapshotManifestEntry{}, err
	}
	svgManifestPath, err := snapshotManifestArtifactPath(cfg.outputDir, paths.SVG)
	if err != nil {
		return snapshotManifestEntry{}, err
	}
	pngManifestPath, err := snapshotManifestArtifactPath(cfg.outputDir, paths.PNG)
	if err != nil {
		return snapshotManifestEntry{}, err
	}
	if err := removeSnapshotArtifactIfPresent(paths.PNG); err != nil {
		return snapshotManifestEntry{}, err
	}
	view, routeID, err := renderSnapshotScenario(scenario, cfg)
	if err != nil {
		return snapshotManifestEntry{}, err
	}
	ansiArtifact, err := writeSnapshotHashedArtifact(paths.ANSI, []byte(view))
	if err != nil {
		return snapshotManifestEntry{}, err
	}
	textArtifact, err := writeSnapshotHashedArtifact(paths.Text, []byte(ansi.Strip(view)))
	if err != nil {
		return snapshotManifestEntry{}, err
	}
	svgArtifact, err := writeSnapshotHashedArtifact(paths.SVG, []byte(renderSnapshotSVG(view, cfg.width, cfg.height)))
	if err != nil {
		return snapshotManifestEntry{}, err
	}
	ansiArtifact.Path = ansiManifestPath
	textArtifact.Path = textManifestPath
	svgArtifact.Path = svgManifestPath
	return snapshotManifestEntry{
		Name:     scenario.Name,
		Viewport: string(cfg.viewport),
		Width:    cfg.width,
		Height:   cfg.height,
		Route:    string(routeID),
		Artifacts: snapshotManifestArtifacts{
			ANSI: ansiArtifact,
			Text: textArtifact,
			SVG:  svgArtifact,
			PNG:  snapshotManifestArtifact{Path: pngManifestPath},
		},
	}, nil
}

func snapshotManifestArtifactPath(outputDir string, artifactPath string) (string, error) {
	relPath, err := filepath.Rel(outputDir, artifactPath)
	if err != nil {
		return "", fmt.Errorf("resolve snapshot artifact path %q relative to %q: %w", artifactPath, outputDir, err)
	}
	relPath = filepath.Clean(relPath)
	if relPath == "." || filepath.IsAbs(relPath) || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("snapshot artifact path %q must stay under %s", artifactPath, outputDir)
	}
	return relPath, nil
}

func removeSnapshotArtifactIfPresent(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func writeSnapshotHashedArtifact(path string, content []byte) (snapshotManifestArtifact, error) {
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return snapshotManifestArtifact{}, err
	}
	sum := sha256.Sum256(content)
	return snapshotManifestArtifact{Path: path, SHA256: hex.EncodeToString(sum[:])}, nil
}

func snapshotPathsForScenario(outputDir string, scenario string, viewport snapshotViewportPreset) snapshotArtifactPaths {
	base := filepath.Join(outputDir, scenario+"."+string(viewport))
	return snapshotArtifactPaths{ANSI: base + ".ansi", Text: base + ".txt", SVG: base + ".svg", PNG: base + ".png"}
}

func writeSnapshotManifest(outputDir string, manifest snapshotManifest) error {
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, "manifest.json"), append(encoded, '\n'), 0o644)
}

func renderSnapshotSVG(ansiView string, cols int, rows int) string {
	layout := newSnapshotSVGLayout(cols, rows)
	buf := cellbuf.NewBuffer(cols, rows)
	cellbuf.SetContent(buf, ansiView)
	var out bytes.Buffer
	writeSnapshotSVGHeader(&out, layout)
	writeSnapshotSVGCells(&out, buf, layout)
	writeSnapshotSVGFooter(&out)
	return out.String()
}

func newSnapshotSVGLayout(cols int, rows int) snapshotSVGLayout {
	layout := snapshotSVGLayout{Cols: cols, Rows: rows, CellWidth: 9, CellHeight: 18, FontSize: 14, Padding: 16}
	layout.Width = cols*layout.CellWidth + layout.Padding*2
	layout.Height = rows*layout.CellHeight + layout.Padding*2
	return layout
}

func writeSnapshotSVGHeader(out *bytes.Buffer, layout snapshotSVGLayout) {
	fmt.Fprintf(out, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, layout.Width, layout.Height, layout.Width, layout.Height)
	out.WriteString("\n")
	fmt.Fprintf(out, `<rect width="100%%" height="100%%" fill="%s"/>`, snapshotDefaultBG())
	out.WriteString("\n")
	fmt.Fprintf(out, `<g font-family="Noto Sans Mono, DejaVu Sans Mono, Menlo, Consolas, monospace" font-size="%d" text-rendering="geometricPrecision">`, layout.FontSize)
	out.WriteString("\n")
}

func writeSnapshotSVGCells(out *bytes.Buffer, buf *cellbuf.Buffer, layout snapshotSVGLayout) {
	for y := 0; y < layout.Rows; y++ {
		for x := 0; x < layout.Cols; x++ {
			writeSnapshotSVGCell(out, buf.Cell(x, y), x, y, layout)
		}
	}
}

func writeSnapshotSVGCell(out *bytes.Buffer, cell *cellbuf.Cell, x int, y int, layout snapshotSVGLayout) {
	if cell == nil || cell.Width == 0 {
		return
	}
	fg, bg := snapshotCellColors(cell)
	px, py := layout.Padding+x*layout.CellWidth, layout.Padding+y*layout.CellHeight
	writeSnapshotSVGCellBackground(out, bg, px, py, cell.Width*layout.CellWidth, layout.CellHeight)
	writeSnapshotSVGCellText(out, cell, fg, px, py+layout.FontSize)
}

func writeSnapshotSVGCellBackground(out *bytes.Buffer, bg string, x int, y int, width int, height int) {
	if bg == snapshotDefaultBG() {
		return
	}
	fmt.Fprintf(out, `<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`, x, y, width, height, bg)
	out.WriteString("\n")
}

func writeSnapshotSVGCellText(out *bytes.Buffer, cell *cellbuf.Cell, fg string, x int, y int) {
	text := cell.String()
	if strings.TrimSpace(text) == "" {
		return
	}
	weight := "400"
	if cell.Style.Attrs&cellbuf.BoldAttr != 0 {
		weight = "700"
	}
	fmt.Fprintf(out, `<text x="%d" y="%d" fill="%s" font-weight="%s">%s</text>`, x, y, fg, weight, escapeSVGText(text))
	out.WriteString("\n")
}

func writeSnapshotSVGFooter(out *bytes.Buffer) {
	out.WriteString("</g>\n</svg>\n")
}

func snapshotCellColors(cell *cellbuf.Cell) (string, string) {
	fg := colorToHex(cell.Style.Fg, snapshotDefaultFG())
	bg := colorToHex(cell.Style.Bg, snapshotDefaultBG())
	if cell.Style.Attrs&cellbuf.ReverseAttr != 0 {
		return bg, fg
	}
	return fg, bg
}

func colorToHex(c color.Color, fallback string) string {
	if c == nil {
		return fallback
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

func escapeSVGText(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "'", "&#39;")
	text = strings.ReplaceAll(text, `"`, "&quot;")
	return text
}

func snapshotDefaultFG() string { return "#d0d0d0" }

func snapshotDefaultBG() string { return "#1c1c1c" }

func withSnapshotColorProfile(fn func() (string, routeID, error)) (string, routeID, error) {
	original := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(original)
	return fn()
}
