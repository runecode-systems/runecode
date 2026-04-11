package policyengine

import (
	"path/filepath"
	"strings"
)

func unwrapLauncherArgv(argv []string) []string {
	idx := 0
	for idx < len(argv) {
		tok := strings.ToLower(filepath.Base(argv[idx]))
		switch tok {
		case "env":
			idx++
			for idx < len(argv) && isEnvAssignmentToken(argv[idx]) {
				idx++
			}
			continue
		case "command", "nohup":
			idx++
			continue
		default:
			return argv[idx:]
		}
	}
	return argv[idx:]
}

func isWorkspaceRelativePath(raw string) bool {
	path := strings.TrimSpace(raw)
	if path == "" {
		return false
	}
	if isCrossPlatformAbsolutePath(path) {
		return false
	}

	clean := filepath.Clean(path)
	normalized := strings.ReplaceAll(clean, "\\", "/")
	return normalized != ".." && !strings.HasPrefix(normalized, "../")
}

func isCrossPlatformAbsolutePath(path string) bool {
	if filepath.IsAbs(path) {
		return true
	}
	if strings.HasPrefix(path, "\\\\") || strings.HasPrefix(path, "\\") {
		return true
	}
	if len(path) >= 2 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' {
		return true
	}
	return false
}
