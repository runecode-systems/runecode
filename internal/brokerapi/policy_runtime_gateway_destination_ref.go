package brokerapi

import (
	"path"
	"strconv"
	"strings"
)

func runtimeDestinationPortMatches(canonicalPort, observedPort *int) bool {
	if canonicalPort != nil {
		return observedPort != nil && *observedPort == *canonicalPort
	}
	const defaultTLSPort = 443
	return observedPort == nil || *observedPort == defaultTLSPort
}

func runtimeDestinationPathPrefixMatches(prefix, observedPath string) bool {
	if strings.TrimSpace(prefix) == "" {
		return true
	}
	observed := runtimeNormalizePath(observedPath)
	normalizedPrefix := runtimeNormalizePath(prefix)
	if normalizedPrefix == "/" {
		return true
	}
	if observed == normalizedPrefix {
		return true
	}
	return strings.HasPrefix(observed, normalizedPrefix+"/")
}

func runtimeNormalizePath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	normalized := path.Clean(trimmed)
	if normalized == "." {
		return "/"
	}
	return normalized
}

func runtimeParseDestinationRef(ref string) (string, *int, string) {
	value := strings.TrimSpace(ref)
	if value == "" {
		return "", nil, ""
	}
	hostPort := value
	pathPart := "/"
	if slash := strings.Index(hostPort, "/"); slash >= 0 {
		pathPart = hostPort[slash:]
		hostPort = hostPort[:slash]
	}
	host := hostPort
	var port *int
	if colon := strings.LastIndex(hostPort, ":"); colon > 0 && colon < len(hostPort)-1 {
		if parsed, err := strconv.Atoi(hostPort[colon+1:]); err == nil && parsed > 0 && parsed <= 65535 {
			host = hostPort[:colon]
			port = &parsed
		}
	}
	if host == "" {
		return "", nil, ""
	}
	return host, port, pathPart
}
