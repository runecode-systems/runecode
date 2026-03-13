package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRunFailsClosedWhenNoFilesExist(t *testing.T) {
	repoRoot := newTempRepo(t, sourceQualityConfig{}, map[string]baselineEntry{})

	err := run([]string{"--root", repoRoot})
	if err == nil || !strings.Contains(err.Error(), "no eligible source files") {
		t.Fatalf("run() error = %v, want no eligible source files", err)
	}
}

func TestRunRejectsRootOutsideWorkspace(t *testing.T) {
	repoRoot := newTempRepo(t, sourceQualityConfig{}, map[string]baselineEntry{})
	outsideRoot := t.TempDir()
	writeJSON(t, filepath.Join(outsideRoot, configFileName), sourceQualityConfig{})
	writeJSON(t, filepath.Join(outsideRoot, baselineFileName), map[string]baselineEntry{})
	writeFile(t, filepath.Join(repoRoot, "cmd/demo/main.go"), "package main\n\nfunc main() {}\n")

	err := run([]string{"--root", outsideRoot})
	if err == nil || !strings.Contains(err.Error(), "root must stay within") {
		t.Fatalf("run() error = %v, want root validation error", err)
	}
}

func TestTierOneJavaScriptRequiresModuleDoc(t *testing.T) {
	repoRoot := newTempRepo(t, sourceQualityConfig{
		RunnerTier1Paths: []string{"runner/scripts/boundary-check.js"},
	}, map[string]baselineEntry{})
	writeFile(t, filepath.Join(repoRoot, "runner/scripts/boundary-check.js"), "const value = 1;\n")

	err := run([]string{"--root", repoRoot})
	if err == nil || !strings.Contains(err.Error(), "source-quality violation") {
		t.Fatalf("run() error = %v, want source-quality violation", err)
	}
}

func TestTierOneJavaScriptAllowsShebangBeforeModuleDoc(t *testing.T) {
	repoRoot := newTempRepo(t, sourceQualityConfig{
		RunnerTier1Paths: []string{"runner/scripts/boundary-check.js"},
	}, map[string]baselineEntry{})
	writeFile(t, filepath.Join(repoRoot, "runner/scripts/boundary-check.js"), "#!/usr/bin/env node\n// boundary guardrail\nconst value = 1;\n")

	if err := run([]string{"--root", repoRoot}); err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}
}

func TestBaselineAllowsOversizedFile(t *testing.T) {
	repoRoot := newTempRepo(t, sourceQualityConfig{}, map[string]baselineEntry{
		"internal/demo/demo.go": {
			Kind:      string(kindSource),
			MaxSloc:   300,
			Rationale: "legacy oversized file",
		},
	})
	writeFile(t, filepath.Join(repoRoot, "internal/demo/demo.go"), goFileWithSourceLines(260))

	if err := run([]string{"--root", repoRoot}); err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}
}

func TestGoFunctionLengthBudgetUsesBodySpan(t *testing.T) {
	repoRoot := newTempRepo(t, sourceQualityConfig{}, map[string]baselineEntry{})
	writeFile(t, filepath.Join(repoRoot, "internal/demo/demo.go"), goFileWithFunctionBodySpan(41))

	err := run([]string{"--root", repoRoot})
	if err == nil || !strings.Contains(err.Error(), "source-quality violation") {
		t.Fatalf("run() error = %v, want function-length violation", err)
	}
}

func TestGoCognitiveComplexityBudget(t *testing.T) {
	repoRoot := newTempRepo(t, sourceQualityConfig{}, map[string]baselineEntry{})
	writeFile(t, filepath.Join(repoRoot, "internal/demo/demo.go"), goFileWithHighCognitiveComplexity())

	violations := collectViolations(t, repoRoot)
	if !hasViolationRule(violations, ruleFunctionCognitiveComplexity) {
		t.Fatalf("collectViolations() = %#v, want %q violation", violations, ruleFunctionCognitiveComplexity)
	}
}

func TestTierOneSuppressionsRequireCheckedInException(t *testing.T) {
	repoRoot := newTempRepo(t, sourceQualityConfig{}, map[string]baselineEntry{})
	writeFile(t, filepath.Join(repoRoot, "internal/demo/demo.go"), "package demo\n\n//nolint:revive // reason\nfunc helper() {}\n")

	err := run([]string{"--root", repoRoot})
	if err == nil || !strings.Contains(err.Error(), "source-quality violation") {
		t.Fatalf("run() error = %v, want tier1 suppression violation", err)
	}
}

func TestTierTwoSuppressionsRequireReason(t *testing.T) {
	repoRoot := newTempRepo(t, sourceQualityConfig{}, map[string]baselineEntry{})
	writeFile(t, filepath.Join(repoRoot, "cmd/demo/main.go"), "package main\n\n//nolint:revive\nfunc main() {}\n")

	err := run([]string{"--root", repoRoot})
	if err == nil || !strings.Contains(err.Error(), "source-quality violation") {
		t.Fatalf("run() error = %v, want suppression-reason violation", err)
	}
}

func collectViolations(t *testing.T, repoRoot string) []violation {
	t.Helper()
	cfg, err := loadRuntimeConfig(repoRoot)
	if err != nil {
		t.Fatalf("loadRuntimeConfig() error = %v", err)
	}
	files, err := collectEligibleFiles(repoRoot, cfg)
	if err != nil {
		t.Fatalf("collectEligibleFiles() error = %v", err)
	}
	violations, err := checkFiles(files, cfg)
	if err != nil {
		t.Fatalf("checkFiles() error = %v", err)
	}
	return violations
}

func hasViolationRule(violations []violation, rule string) bool {
	for _, violation := range violations {
		if violation.rule == rule {
			return true
		}
	}
	return false
}

func TestCommentedOutCodeHeuristicsAvoidSuppressionAndAnnotations(t *testing.T) {
	file := fileInfo{relPath: "internal/demo/demo.go"}
	content := strings.Join([]string{
		"//nolint:revive // reason",
		"// NOTE: keep the branch fail-closed.",
		"// TODO: add more edge cases.",
		"// if err != nil {",
	}, "\n")

	violations := checkCommentedOutCode(file, content)
	if len(violations) != 1 || violations[0].rule != ruleCommentedOutCode {
		t.Fatalf("checkCommentedOutCode() violations = %#v, want one commented-out-code violation", violations)
	}
}

func TestCountSourceLines(t *testing.T) {
	content := strings.Join([]string{
		"package demo",
		"",
		"// line comment",
		"/* block comment */",
		"var first = 1",
		"/*",
		"comment",
		"*/",
		"var second = 2",
	}, "\n")

	if got := countSourceLines(content); got != 3 {
		t.Fatalf("countSourceLines() = %d, want 3", got)
	}
}

func TestIsGeneratedFile(t *testing.T) {
	content := strings.Join([]string{
		"// Copyright 2026",
		"// Code generated by tool; DO NOT EDIT.",
		"package demo",
	}, "\n")

	if !isGeneratedFile(content) {
		t.Fatal("isGeneratedFile() = false, want true")
	}
}

func TestHasLeadingModuleComment(t *testing.T) {
	content := strings.Join([]string{
		"#!/usr/bin/env node",
		"// boundary guardrail",
		"const value = 1;",
	}, "\n")

	if !hasLeadingModuleComment(content) {
		t.Fatal("hasLeadingModuleComment() = false, want true")
	}
}

func TestClassifyKindTreatsTestsDirectoryByPathSegment(t *testing.T) {
	if got := classifyKind("runner/src/tests-utils/example.ts", languageTS); got != kindSource {
		t.Fatalf("classifyKind() = %q, want %q", got, kindSource)
	}
	if got := classifyKind("runner/src/tests/example.ts", languageTS); got != kindTest {
		t.Fatalf("classifyKind() = %q, want %q", got, kindTest)
	}
}

func newTempRepo(t *testing.T, cfg sourceQualityConfig, baseline map[string]baselineEntry) string {
	t.Helper()
	repoRoot := t.TempDir()
	t.Chdir(repoRoot)
	writeJSON(t, filepath.Join(repoRoot, configFileName), cfg)
	writeJSON(t, filepath.Join(repoRoot, baselineFileName), baseline)
	return repoRoot
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	contents, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	writeFile(t, path, string(contents))
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func goFileWithSourceLines(lines int) string {
	parts := []string{"package demo", ""}
	for i := 0; i < lines; i++ {
		parts = append(parts, "var value"+strconv.Itoa(i)+" = "+strconv.Itoa(i))
	}
	parts = append(parts, "")
	return strings.Join(parts, "\n")
}

func goFileWithFunctionBodySpan(lines int) string {
	parts := []string{"package demo", "", "func helper() {"}
	for i := 0; i < lines-2; i++ {
		parts = append(parts, "\tprintln(\"line\")")
	}
	parts = append(parts, "}", "")
	return strings.Join(parts, "\n")
}

func goFileWithHighCognitiveComplexity() string {
	return strings.Join([]string{
		"package demo",
		"",
		"func helper(values []int) int {",
		"\ttotal := 0",
		"OUT:",
		"\tfor _, value := range values {",
		"\t\tif value > 0 {",
		"\t\t\tfor i := 0; i < value; i++ {",
		"\t\t\t\tif i%2 == 0 {",
		"\t\t\t\t\tcontinue OUT",
		"\t\t\t\t}",
		"\t\t\t}",
		"\t\t}",
		"\t\ttotal += value",
		"\t}",
		"\treturn total",
		"}",
		"",
	}, "\n")
}
