package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderDirectoryEmptyAndSelected(t *testing.T) {
	if got := renderDirectory("Runs", nil, 0); !strings.Contains(got, "EMPTY") {
		t.Fatalf("expected empty state, got %q", got)
	}
	got := renderDirectory("Runs", []string{"run-1", "run-2"}, 1)
	if !strings.Contains(got, "> run-2") {
		t.Fatalf("expected selected marker in %q", got)
	}
}

func TestRenderBoundedListAppliesWidthBoundingAndSelection(t *testing.T) {
	got := renderBoundedList(boundedListSpec{
		Rows: []boundedListRow{
			{Text: "  alpha-long", Selectable: true},
			{Text: "  beta", Selectable: true},
		},
		Selected: 0,
		Width:    8,
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two rendered lines, got %d: %q", len(lines), got)
	}
	if lines[0] != "alp..." {
		t.Fatalf("expected first line clipped to width with ascii ellipsis, got %q", lines[0])
	}
	if lines[1] != "  beta" {
		t.Fatalf("expected second line unchanged, got %q", lines[1])
	}
}

func TestRenderBoundedListPreservesGapRows(t *testing.T) {
	got := renderBoundedList(boundedListSpec{
		Rows: []boundedListRow{
			{Text: "row-1", Selectable: true},
			{Text: "", Selectable: false},
			{Text: "row-2", Selectable: true},
		},
		Selected:     1,
		PreserveGaps: true,
	})
	if !strings.Contains(got, "row-1\n\nrow-2") {
		t.Fatalf("expected blank gap row preserved, got %q", got)
	}
}

func TestRenderBoundedListHeightUsesGapMarkersAroundSelection(t *testing.T) {
	got := renderBoundedList(boundedListSpec{
		Rows: []boundedListRow{
			{Text: "row-0", Selectable: true},
			{Text: "row-1", Selectable: true},
			{Text: "row-2", Selectable: true},
			{Text: "row-3", Selectable: true},
			{Text: "row-4", Selectable: true},
			{Text: "row-5", Selectable: true},
			{Text: "row-6", Selectable: true},
		},
		Selected:  3,
		Height:    5,
		GapMarker: "...",
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected bounded height 5, got %d: %q", len(lines), got)
	}
	if lines[0] != "..." || lines[len(lines)-1] != "..." {
		t.Fatalf("expected top/bottom gap markers, got %q", got)
	}
	if !strings.Contains(got, "row-3") {
		t.Fatalf("expected selected neighborhood to include row-3, got %q", got)
	}
}

func TestRenderDirectorySelectedMarkerSurvivesWidthClipping(t *testing.T) {
	got := renderDirectory("Runs", []string{"run-1", "run-2-with-a-very-long-suffix-that-will-be-clipped"}, 1)
	if !strings.Contains(got, "> run-2-with-a-very-long-suffix-that-will-be-clipped") {
		t.Fatalf("expected selected marker before clipping, got %q", got)
	}

	clipped := lipgloss.NewStyle().Width(12).MaxWidth(12).Render(got)
	if !strings.Contains(clipped, ">") {
		t.Fatalf("expected selected marker to survive clipping, got %q", clipped)
	}
	for _, line := range strings.Split(clipped, "\n") {
		if lipgloss.Width(line) > 12 {
			t.Fatalf("expected clipped directory line width <= 12, got %d in %q", lipgloss.Width(line), line)
		}
	}
}

func TestRenderModeSwitchTabs(t *testing.T) {
	got := renderModeSwitchTabs([]string{"rendered", "raw", "structured"}, "raw")
	if !strings.Contains(got, "[RAW]") {
		t.Fatalf("expected active tab marker, got %q", got)
	}
}

func TestLongFormViewportRendersContent(t *testing.T) {
	got := renderLongFormViewport("line one\nline two", 20, 4)
	if !strings.Contains(got, "line one") {
		t.Fatalf("expected viewport content in %q", got)
	}
}

func TestRenderInspectorShellIncludesCanonicalSections(t *testing.T) {
	got := renderInspectorShell(inspectorShellSpec{
		Title:          "Session inspector",
		Summary:        "session summary",
		Identity:       "session=session-1 workspace=ws-1",
		Status:         "status=active",
		Badges:         []string{"[status:active]"},
		References:     []inspectorReference{{Label: "runs", Items: []inspectorReferenceItem{{Label: "run-1"}}}},
		LocalActions:   []routeActionItem{{Label: "jump:runs"}, {Label: "copy:session_id"}},
		ModeTabs:       []string{string(presentationRendered), string(presentationRaw), string(presentationStructured)},
		ActiveMode:     string(presentationRaw),
		ContentKind:    inspectorContentTranscript,
		ContentLabel:   "transcript",
		Content:        "hello",
		ViewportWidth:  96,
		ViewportHeight: 12,
		CopyActions:    []routeCopyAction{{ID: "session_id", Label: "session id", Text: "session-1"}},
	})
	mustContainAll(t, got,
		"Overview",
		"Summary: session summary",
		"Identity: session=session-1 workspace=ws-1",
		"Status: status=active",
		"Linked references",
		"Linked runs: run-1",
		"Actions",
		"Local actions: jump:runs | copy:session_id",
		"Copy actions: session id",
		"Summary → detail: RAW mode",
		"Detail viewport",
		"[RAW]",
		"[transcript viewport 96x12]",
	)
}

func TestFormatInspectorLongFormDiffAddsSummary(t *testing.T) {
	got := formatInspectorLongForm(inspectorContentDiff, "@@\n- old\n+ new\n unchanged")
	mustContainAll(t, got,
		"Diff summary: lines=4 additions=1 deletions=1",
		"+ new",
		"- old",
	)
}

func TestFormatInspectorLongFormMarkdownImprovesReadability(t *testing.T) {
	got := formatInspectorLongForm(inspectorContentMarkdown, "# Header\n- one\n* two")
	mustContainAll(t, got,
		"Markdown reading view:",
		"§ Header",
		"• one",
		"• two",
	)
}

func TestFormatInspectorLongFormStructuredNumbersFields(t *testing.T) {
	got := formatInspectorLongForm(inspectorContentStructured, "turn_count=2\nmessage_count=5")
	mustContainAll(t, got,
		"Structured reading view:",
		"1) turn_count = 2",
		"2) message_count = 5",
	)
}

func TestLongFormDocumentStatePersistsViewportAndResetsOnDocumentChange(t *testing.T) {
	doc := newLongFormDocumentState()
	doc.Resize(96, 8)
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11"
	doc.SetDocument(workbenchObjectRef{Kind: "run", ID: "run-1"}, inspectorContentLog, "log", content)
	doc.Scroll(3)
	if got := doc.Render(); !strings.Contains(got, "offset=3") {
		t.Fatalf("expected offset 3 after scroll, got %q", got)
	}

	doc.SetDocument(workbenchObjectRef{Kind: "run", ID: "run-1"}, inspectorContentLog, "log", content)
	if got := doc.Render(); !strings.Contains(got, "offset=3") {
		t.Fatalf("expected same-document offset persistence, got %q", got)
	}

	doc.SetDocument(workbenchObjectRef{Kind: "run", ID: "run-2"}, inspectorContentLog, "log", content)
	if got := doc.Render(); !strings.Contains(got, "offset=0") {
		t.Fatalf("expected offset reset on document swap, got %q", got)
	}
}
