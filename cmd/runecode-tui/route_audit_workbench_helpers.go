package main

import (
	"fmt"
	"strings"

	"github.com/runecode-systems/runecode/internal/brokerapi"
)

func auditEventLabel(eventType string) string {
	switch strings.TrimSpace(eventType) {
	case "run_state":
		return "Run state update"
	case "approval":
		return "Approval event"
	case "audit_receipt":
		return "Anchor receipt recorded"
	case "artifact":
		return "Artifact evidence updated"
	case "verification":
		return "Verification record"
	default:
		if strings.TrimSpace(eventType) == "" {
			return "Audit record"
		}
		words := strings.Fields(strings.ReplaceAll(strings.TrimSpace(eventType), "_", " "))
		for i, word := range words {
			if word == "id" {
				words[i] = "ID"
				continue
			}
			if word == "" {
				continue
			}
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
		return strings.Join(words, " ")
	}
}

func auditVerificationCue(status string, reasonCodes []string) string {
	status = valueOrNA(strings.TrimSpace(status))
	switch status {
	case "ok":
		return "verification healthy"
	case "degraded":
		return "verification degraded"
	case "failed", "invalid":
		return "verification failed"
	default:
		return "verification " + status
	}
}

func renderAuditNextActionCue(verify *brokerapi.AuditVerificationGetResponse, finalize *brokerapi.AuditFinalizeVerifyResponse) string {
	if verify == nil {
		return "reload verification, then review findings and anchoring options"
	}
	if verify.Summary.CurrentlyDegraded || strings.EqualFold(strings.TrimSpace(verify.Summary.AnchoringStatus), "degraded") {
		return "review findings first, then finalize/verify or export evidence for offline review"
	}
	if finalize == nil || !strings.EqualFold(strings.TrimSpace(finalize.ActionStatus), "ok") {
		return "finalize/verify if needed, then export or anchor the latest verified segment"
	}
	return "use export copy for offline verification or anchor the latest verified segment"
}

func renderAuditAnchorWorkbenchLine(anchoring, exportCopy bool, statusText string) string {
	if anchoring {
		return "Anchoring: request in flight; broker is preparing anchored evidence"
	}
	copyState := "export copy off"
	if exportCopy {
		copyState = "export copy on"
	}
	status := strings.TrimSpace(statusText)
	if status == "" {
		return "Anchoring: ready • " + copyState + " • use receipt for offline verification when available"
	}
	return fmt.Sprintf("Anchoring: %s • %s", status, copyState)
}
