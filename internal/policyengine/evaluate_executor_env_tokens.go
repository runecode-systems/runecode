package policyengine

import "strings"

func isEnvAssignmentToken(token string) bool {
	t := strings.TrimSpace(token)
	if t == "" {
		return false
	}
	eq := strings.IndexByte(t, '=')
	if eq <= 0 {
		return false
	}
	return isEnvAssignmentName(t[:eq])
}

func isEnvAssignmentName(name string) bool {
	for i, r := range name {
		if i == 0 {
			if !isEnvAssignmentStartRune(r) {
				return false
			}
			continue
		}
		if !isEnvAssignmentContinueRune(r) {
			return false
		}
	}
	return true
}

func isEnvAssignmentStartRune(r rune) bool {
	return r == '_' || isASCIIAlpha(r)
}

func isEnvAssignmentContinueRune(r rune) bool {
	return isEnvAssignmentStartRune(r) || isASCIIDigit(r)
}

func isASCIIAlpha(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
