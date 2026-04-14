package brokerapi

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func ParseDataClass(value string) (artifacts.DataClass, error) {
	class := artifacts.DataClass(value)
	switch class {
	case artifacts.DataClassSpecText,
		artifacts.DataClassUnapprovedFileExcerpts,
		artifacts.DataClassApprovedFileExcerpts,
		artifacts.DataClassDiffs,
		artifacts.DataClassBuildLogs,
		artifacts.DataClassAuditEvents,
		artifacts.DataClassAuditVerificationReport,
		artifacts.DataClassWebQuery,
		artifacts.DataClassWebCitations:
		return class, nil
	default:
		return "", fmt.Errorf("unsupported data class %q", value)
	}
}
