package main

import "regexp"

var secretLinePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(api[_-]?key\s*[:=]\s*)([^\s,;]+)`),
	regexp.MustCompile(`(?i)(token\s*[:=]\s*)([^\s,;]+)`),
	regexp.MustCompile(`(?i)(password\s*[:=]\s*)([^\s,;]+)`),
	regexp.MustCompile(`(?i)(secret\s*[:=]\s*)([^\s,;]+)`),
	regexp.MustCompile(`(?i)([A-Z_]*(?:SECRET|TOKEN|KEY|PASSWORD|CREDENTIAL)[A-Z_]*\s*[=:]\s*)([^\s,;"'\n]+)`),
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
