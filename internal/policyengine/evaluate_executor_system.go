package policyengine

import (
	"path/filepath"
	"strings"
	"unicode"
)

func isSystemModifyingArgv(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	for _, token := range argv {
		if tokenIsSystemModifying(token) {
			return true
		}
	}
	return false
}

func tokenIsSystemModifying(token string) bool {
	lower := strings.ToLower(strings.TrimSpace(token))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "/etc/") || strings.Contains(lower, "c:\\windows") {
		return true
	}
	base := strings.ToLower(filepath.Base(lower))
	if _, ok := systemModifyingExecutableNames[base]; ok {
		return true
	}
	for _, candidate := range splitCommandLikeToken(lower) {
		candidateBase := strings.ToLower(filepath.Base(candidate))
		if _, ok := systemModifyingExecutableNames[candidateBase]; ok {
			return true
		}
	}
	return false
}

func splitCommandLikeToken(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune("'\"`;|&(){}[]<>,", r)
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, strings.Trim(trimmed, "`"))
	}
	return out
}
