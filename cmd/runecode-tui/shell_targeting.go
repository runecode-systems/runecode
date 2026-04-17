package main

import (
	"os"
	"strings"
)

func logicalBrokerTargetKey() string {
	alias := strings.TrimSpace(os.Getenv("RUNECODE_TUI_BROKER_TARGET"))
	if alias == "" {
		return localAPIFamily + ":local-default"
	}
	alias = normalizeBrokerTargetAlias(alias)
	return localAPIFamily + ":" + alias
}

func normalizeBrokerTargetAlias(alias string) string {
	const maxAliasLen = 128
	b := strings.Builder{}
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(alias)) {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum || r == '-' || r == '_' {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
		if b.Len() >= maxAliasLen {
			break
		}
	}
	normalized := strings.Trim(b.String(), "-")
	if normalized == "" {
		return "local-default"
	}
	return normalized
}
