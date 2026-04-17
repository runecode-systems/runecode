package main

import (
	"strings"
	"testing"
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
