package projectsubstrate

import (
	"fmt"
	"strconv"
	"strings"
)

func compareVersion(actual, want string) (int, error) {
	a, err := parseVersion(actual)
	if err != nil {
		return 0, err
	}
	b, err := parseVersion(want)
	if err != nil {
		return 0, err
	}
	if cmp := compareCoreVersion(a, b); cmp != 0 {
		return cmp, nil
	}
	return comparePreRelease(a.preRelease, b.preRelease), nil
}

type parsedVersion struct {
	major      int
	minor      int
	patch      int
	preRelease []string
}

func compareCoreVersion(a, b parsedVersion) int {
	if cmp := compareVersionNumber(a.major, b.major); cmp != 0 {
		return cmp
	}
	if cmp := compareVersionNumber(a.minor, b.minor); cmp != 0 {
		return cmp
	}
	return compareVersionNumber(a.patch, b.patch)
}

func compareVersionNumber(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func parseVersion(value string) (parsedVersion, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return parsedVersion{}, fmt.Errorf("empty version")
	}
	parts := strings.SplitN(v, "-", 2)
	core, err := parseCoreVersion(parts[0], value)
	if err != nil {
		return parsedVersion{}, err
	}
	if len(parts) == 2 {
		preRelease, err := parsePreRelease(parts[1], value)
		if err != nil {
			return parsedVersion{}, err
		}
		core.preRelease = preRelease
	}
	return core, nil
}

func parseCoreVersion(coreValue, raw string) (parsedVersion, error) {
	core := strings.Split(coreValue, ".")
	if len(core) != 3 {
		return parsedVersion{}, fmt.Errorf("invalid version %q", raw)
	}
	major, err := strconv.Atoi(core[0])
	if err != nil {
		return parsedVersion{}, fmt.Errorf("invalid major version %q", raw)
	}
	if !isCanonicalNumericIdentifier(core[0]) {
		return parsedVersion{}, fmt.Errorf("invalid major version %q", raw)
	}
	minor, err := strconv.Atoi(core[1])
	if err != nil {
		return parsedVersion{}, fmt.Errorf("invalid minor version %q", raw)
	}
	if !isCanonicalNumericIdentifier(core[1]) {
		return parsedVersion{}, fmt.Errorf("invalid minor version %q", raw)
	}
	patch, err := strconv.Atoi(core[2])
	if err != nil {
		return parsedVersion{}, fmt.Errorf("invalid patch version %q", raw)
	}
	if !isCanonicalNumericIdentifier(core[2]) {
		return parsedVersion{}, fmt.Errorf("invalid patch version %q", raw)
	}
	return parsedVersion{major: major, minor: minor, patch: patch}, nil
}

func parsePreRelease(value, raw string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("invalid pre-release %q", raw)
	}
	parts := strings.Split(value, ".")
	for _, part := range parts {
		if !isValidPreReleaseIdentifier(part) {
			return nil, fmt.Errorf("invalid pre-release %q", raw)
		}
	}
	return parts, nil
}

func isCanonicalNumericIdentifier(value string) bool {
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	if len(value) > 1 && value[0] == '0' {
		return false
	}
	return true
}

func isValidPreReleaseIdentifier(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && r != '-' {
			return false
		}
	}
	if isAllDigits(value) {
		return isCanonicalNumericIdentifier(value)
	}
	return true
}

func isAllDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func comparePreRelease(a, b []string) int {
	if cmp := comparePreReleasePresence(a, b); cmp != 0 {
		return cmp
	}
	for i := 0; i < maxPreReleaseLen(a, b); i++ {
		if cmp, done := comparePreReleaseBounds(a, b, i); done {
			return cmp
		}
		if cmp := comparePreReleaseIdentifier(a[i], b[i]); cmp != 0 {
			return cmp
		}
	}
	return 0
}

func comparePreReleasePresence(a, b []string) int {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	if len(a) == 0 {
		return 1
	}
	if len(b) == 0 {
		return -1
	}
	return 0
}

func maxPreReleaseLen(a, b []string) int {
	if len(a) > len(b) {
		return len(a)
	}
	return len(b)
}

func comparePreReleaseBounds(a, b []string, i int) (int, bool) {
	if i >= len(a) {
		return -1, true
	}
	if i >= len(b) {
		return 1, true
	}
	return 0, false
}

func comparePreReleaseIdentifier(a, b string) int {
	ai, aErr := strconv.Atoi(a)
	bi, bErr := strconv.Atoi(b)
	if aErr == nil && bErr == nil {
		return compareVersionNumber(ai, bi)
	}
	if aErr == nil {
		return -1
	}
	if bErr == nil {
		return 1
	}
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
