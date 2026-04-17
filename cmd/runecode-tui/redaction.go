package main

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var secretLinePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(api[_-]?key\s*[:=]\s*)([^\s,;]+)`),
	regexp.MustCompile(`(?i)(token\s*[:=]\s*)([^\s,;]+)`),
	regexp.MustCompile(`(?i)(password\s*[:=]\s*)([^\s,;]+)`),
	regexp.MustCompile(`(?i)(secret\s*[:=]\s*)([^\s,;]+)`),
	regexp.MustCompile(`(?i)([A-Z_]*(?:SECRET|TOKEN|KEY|PASSWORD|CREDENTIAL)[A-Z_]*\s*[=:]\s*)([^\s,;"'\n]+)`),
	regexp.MustCompile(`(?i)(export\s+[A-Z_]*(?:SECRET|TOKEN|KEY|PASSWORD|CREDENTIAL)[A-Z_]*\s*=\s*)([^\s,;"'\n]+)`),
	regexp.MustCompile(`(?i)("(?:api[_-]?key|token|password|secret|credential|access[_-]?key)"\s*:\s*")([^"]+)(")`),
	regexp.MustCompile(`(?i)('(?:api[_-]?key|token|password|secret|credential|access[_-]?key)'\s*:\s*')([^']+)(')`),
	regexp.MustCompile(`(?i)(authorization\s*[:=]\s*bearer\s+)([^\s,;]+)`),
}

func redactSecrets(text string) string {
	redacted := text
	for _, pattern := range secretLinePatterns {
		replacement := `${1}[REDACTED]`
		if pattern.NumSubexp() >= 3 {
			replacement = `${1}[REDACTED]${3}`
		}
		redacted = pattern.ReplaceAllString(redacted, replacement)
	}
	return redacted
}

func sanitizeUIText(text string) string {
	text = strings.TrimSpace(redactSecrets(text))
	if text == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range text {
		if r == '\t' || (unicode.IsPrint(r) && r != 0x1b && r != '\n' && r != '\r') {
			b.WriteRune(r)
		}
	}
	sanitized := strings.TrimSpace(b.String())
	if len(sanitized) > 512 {
		cutoff := 512
		for cutoff > 0 && !utf8.RuneStart(sanitized[cutoff]) {
			cutoff--
		}
		return sanitized[:cutoff] + "..."
	}
	return sanitized
}
