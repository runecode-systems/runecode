package main

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	annotationPrefixes = []string{
		"NOTE:",
		"TODO:",
		"FIXME:",
		"HACK:",
		"XXX:",
		"SECURITY:",
		"INVARIANT:",
		"WARNING:",
		"BUG:",
	}
	codeAssignmentPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*\s*(?::=|=)\s*\S+`)
	codeCallPattern       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*\([^)]*\)$`)
	nolintReasonPattern   = regexp.MustCompile(`^//\s*nolint(?::[A-Za-z0-9_, -]+)?\s*(?://\s*\S+|--\s*\S+)`)
	eslintReasonPattern   = regexp.MustCompile(`^(?://|/\*|\*)\s*eslint-disable(?:-next-line|-line)?(?:\s+[A-Za-z0-9_, -]+)?\s+--\s*\S+`)
)

func checkSuppressions(file fileInfo, content string, cfg runtimeConfig) []violation {
	violations := make([]violation, 0)
	for lineNumber, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !isCommentLine(trimmed) || !isSuppressionComment(trimmed) {
			continue
		}

		context := "line " + strconv.Itoa(lineNumber+1)
		violations = append(violations, suppressionViolationsForLine(file, trimmed, context, cfg)...)
	}
	return violations
}

func suppressionViolationsForLine(file fileInfo, trimmed, context string, cfg runtimeConfig) []violation {
	if file.tier == tierOne {
		if _, allowed := cfg.tier1SuppressionExceptions[file.relPath]; !allowed {
			return []violation{{
				rule:        ruleTierOneSuppression,
				path:        file.relPath,
				context:     context,
				observed:    trimmed,
				expected:    "no inline suppression in Tier 1 files",
				remediation: "move the exception into checked-in checker configuration with a rationale",
			}}
		}
		return nil
	}

	violations := make([]violation, 0)
	if strings.Contains(trimmed, "nolint") && !nolintReasonPattern.MatchString(trimmed) {
		violations = append(violations, violation{
			rule:        ruleSuppressionReason,
			path:        file.relPath,
			context:     context,
			observed:    trimmed,
			expected:    "//nolint includes a specific reason",
			remediation: "append a reason using '// reason' or '-- reason'",
		})
	}
	if strings.Contains(trimmed, "eslint-disable") && !eslintReasonPattern.MatchString(trimmed) {
		violations = append(violations, violation{
			rule:        ruleSuppressionReason,
			path:        file.relPath,
			context:     context,
			observed:    trimmed,
			expected:    "eslint-disable includes '-- reason'",
			remediation: "append a reason using '-- reason'",
		})
	}
	return violations
}

func checkCommentedOutCode(file fileInfo, content string) []violation {
	violations := make([]violation, 0)
	for lineNumber, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !isCommentLine(trimmed) || isSuppressionComment(trimmed) {
			continue
		}

		commentText := extractCommentText(trimmed)
		if !looksLikeCommentedCode(commentText) {
			continue
		}

		violations = append(violations, violation{
			rule:        ruleCommentedOutCode,
			path:        file.relPath,
			context:     "line " + strconv.Itoa(lineNumber+1),
			observed:    trimmed,
			expected:    "comments explain rationale rather than preserve code",
			remediation: "remove the commented-out code or move the rationale into prose",
		})
	}
	return violations
}

func isCommentLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "* ")
}

func isSuppressionComment(trimmed string) bool {
	return strings.Contains(trimmed, "nolint") || strings.Contains(trimmed, "eslint-disable")
}

func extractCommentText(trimmed string) string {
	switch {
	case strings.HasPrefix(trimmed, "//"):
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
	case strings.HasPrefix(trimmed, "/*"):
		text := strings.TrimSpace(strings.TrimPrefix(trimmed, "/*"))
		return strings.TrimSpace(strings.TrimSuffix(text, "*/"))
	case strings.HasPrefix(trimmed, "* "):
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "* "))
	default:
		return trimmed
	}
}

func looksLikeCommentedCode(commentText string) bool {
	if commentText == "" || hasAllowedAnnotationPrefix(commentText) {
		return false
	}

	for _, prefix := range []string{"package ", "import ", "func ", "type ", "var ", "const ", "let ", "class ", "interface ", "export ", "module.exports", "require("} {
		if strings.HasPrefix(commentText, prefix) {
			return true
		}
	}

	if codeAssignmentPattern.MatchString(commentText) || codeCallPattern.MatchString(commentText) {
		return true
	}

	if strings.HasPrefix(commentText, "if ") || strings.HasPrefix(commentText, "for ") || strings.HasPrefix(commentText, "switch ") || strings.HasPrefix(commentText, "case ") {
		return strings.ContainsAny(commentText, "{}()[]=<>") || strings.Contains(commentText, ":=")
	}

	return false
}

func hasAllowedAnnotationPrefix(commentText string) bool {
	upper := strings.ToUpper(commentText)
	for _, prefix := range annotationPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	return false
}
