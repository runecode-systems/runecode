package main

import (
	"fmt"
	"sort"
	"strings"
)

const (
	ruleFileSLOC                    = "file-sloc"
	ruleModuleDoc                   = "module-doc"
	ruleSuppressionReason           = "suppression-reason"
	ruleTierOneSuppression          = "tier1-suppression"
	ruleCommentedOutCode            = "commented-out-code"
	ruleFunctionLength              = "function-length"
	ruleFunctionCognitiveComplexity = "function-cognitive-complexity"
)

type violation struct {
	rule        string
	path        string
	context     string
	observed    string
	expected    string
	remediation string
}

func (v violation) format() string {
	parts := []string{v.path}
	if v.context != "" {
		parts = append(parts, v.context)
	}
	if v.observed != "" || v.expected != "" {
		parts = append(parts, fmt.Sprintf("observed=%s expected=%s", v.observed, v.expected))
	}
	if v.remediation != "" {
		parts = append(parts, fmt.Sprintf("remediation=%s", v.remediation))
	}
	return strings.Join(parts, " | ")
}

func checkFiles(files []fileInfo, cfg runtimeConfig) ([]violation, error) {
	violations := make([]violation, 0)
	for _, file := range files {
		fileViolations, err := checkFile(file, cfg)
		if err != nil {
			return nil, err
		}
		violations = append(violations, fileViolations...)
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].path == violations[j].path {
			if violations[i].rule == violations[j].rule {
				return violations[i].context < violations[j].context
			}
			return violations[i].rule < violations[j].rule
		}
		return violations[i].path < violations[j].path
	})

	return violations, nil
}

func checkFile(file fileInfo, cfg runtimeConfig) ([]violation, error) {
	violations := make([]violation, 0)
	violations = append(violations, checkFileBudget(file, file.content, cfg)...)
	violations = append(violations, checkModuleDoc(file, file.content)...)
	violations = append(violations, checkSuppressions(file, file.content, cfg)...)
	violations = append(violations, checkCommentedOutCode(file, file.content)...)

	if file.language == languageGo {
		goViolations, err := checkGoFunctionLengths(file, file.content, cfg)
		if err != nil {
			return nil, err
		}
		violations = append(violations, goViolations...)
	}

	return violations, nil
}

func checkFileBudget(file fileInfo, content string, cfg runtimeConfig) []violation {
	sloc := countSourceLines(content)
	limit := defaultSlocLimit(file)
	if entry, ok := cfg.baseline[file.relPath]; ok && entry.MaxSloc > limit {
		limit = entry.MaxSloc
	}
	if sloc <= limit {
		return nil
	}

	return []violation{{
		rule:        ruleFileSLOC,
		path:        file.relPath,
		context:     string(file.tier) + " " + string(file.kind),
		observed:    fmt.Sprintf("%d SLOC", sloc),
		expected:    fmt.Sprintf("<= %d SLOC", limit),
		remediation: "split the file or add a reviewed baseline entry",
	}}
}

func defaultSlocLimit(file fileInfo) int {
	if file.kind == kindTest {
		if file.tier == tierOne {
			return 500
		}
		return 800
	}

	if file.tier == tierOne {
		return 250
	}
	return 400
}

func checkModuleDoc(file fileInfo, content string) []violation {
	if file.kind == kindTest || file.tier != tierOne {
		return nil
	}
	if file.language != languageJS && file.language != languageTS {
		return nil
	}
	if hasLeadingModuleComment(content) {
		return nil
	}

	return []violation{{
		rule:        ruleModuleDoc,
		path:        file.relPath,
		observed:    "missing top-of-file module comment",
		expected:    "Tier 1 JS/TS modules start with a module doc",
		remediation: "add a short top-of-file module doc that explains purpose and boundary role",
	}}
}

func hasLeadingModuleComment(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#!") {
			continue
		}
		return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*")
	}
	return false
}

func countSourceLines(content string) int {
	count := 0
	inBlockComment := false

	for _, rawLine := range strings.Split(content, "\n") {
		line, nextBlockComment := stripCommentOnlySegments(strings.TrimSpace(rawLine), inBlockComment)
		inBlockComment = nextBlockComment

		if line != "" {
			count++
		}
	}

	return count
}

func stripCommentOnlySegments(line string, inBlockComment bool) (string, bool) {
	for {
		switch {
		case inBlockComment:
			nextLine, nextBlockComment, advanced := consumeBlockComment(line)
			if !advanced {
				return "", true
			}
			line = nextLine
			inBlockComment = nextBlockComment
		case line == "" || strings.HasPrefix(line, "//"):
			return "", false
		case strings.HasPrefix(line, "/*"):
			nextLine, nextBlockComment, _ := consumeBlockComment(strings.TrimSpace(line[2:]))
			line = nextLine
			inBlockComment = nextBlockComment
		default:
			return line, false
		}
	}
}

func consumeBlockComment(line string) (string, bool, bool) {
	end := strings.Index(line, "*/")
	if end == -1 {
		return "", true, false
	}

	return strings.TrimSpace(line[end+2:]), false, true
}
