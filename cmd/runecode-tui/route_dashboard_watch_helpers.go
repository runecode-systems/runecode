package main

import (
	"fmt"
	"strings"

	"github.com/runecode-systems/runecode/internal/brokerapi"
)

func summarizeRunWatchEvents(events []brokerapi.RunWatchEvent) watchFamilySummary {
	s := watchFamilySummary{family: "run_watch", eventCount: len(events), lastStatus: "ok"}
	for _, event := range events {
		s.lastEventType = event.EventType
		if event.Run != nil {
			s.lastSubject = event.Run.RunID
		}
		if event.Terminal {
			s.terminalCount++
			s.lastStatus = event.TerminalStatus
		}
		if event.Error != nil {
			s.errorCount++
			s.lastStatus = "error"
		}
		switch event.EventType {
		case "run_watch_snapshot":
			s.snapshotCount++
		case "run_watch_upsert":
			s.upsertCount++
		}
	}
	return s
}

func renderLiveActivityFeed(feed []shellLiveActivityEntry) string {
	if len(feed) == 0 {
		return "  feed: waiting for shell watch manager"
	}
	lines := make([]string, 0, len(feed)+1)
	lines = append(lines, "  feed:")
	for i := len(feed) - 1; i >= 0; i-- {
		e := feed[i]
		lines = append(lines, fmt.Sprintf("    %s event=%s subject=%s status=%s", infoBadge(sanitizeUIText(e.Family)), valueOrNA(sanitizeUIText(e.EventType)), valueOrNA(sanitizeUIText(e.Subject)), valueOrNA(sanitizeUIText(e.Status))))
	}
	return strings.Join(lines, "\n")
}

func summarizeApprovalWatchEvents(events []brokerapi.ApprovalWatchEvent) watchFamilySummary {
	s := watchFamilySummary{family: "approval_watch", eventCount: len(events), lastStatus: "ok"}
	for _, event := range events {
		s.lastEventType = event.EventType
		if event.Approval != nil {
			s.lastSubject = event.Approval.ApprovalID
		}
		if event.Terminal {
			s.terminalCount++
			s.lastStatus = event.TerminalStatus
		}
		if event.Error != nil {
			s.errorCount++
			s.lastStatus = "error"
		}
		switch event.EventType {
		case "approval_watch_snapshot":
			s.snapshotCount++
		case "approval_watch_upsert":
			s.upsertCount++
		}
	}
	return s
}

func summarizeSessionWatchEvents(events []brokerapi.SessionWatchEvent) watchFamilySummary {
	s := watchFamilySummary{family: "session_watch", eventCount: len(events), lastStatus: "ok"}
	for _, event := range events {
		s.lastEventType = event.EventType
		if event.Session != nil {
			s.lastSubject = event.Session.Identity.SessionID
		}
		if event.Terminal {
			s.terminalCount++
			s.lastStatus = event.TerminalStatus
		}
		if event.Error != nil {
			s.errorCount++
			s.lastStatus = "error"
		}
		switch event.EventType {
		case "session_watch_snapshot":
			s.snapshotCount++
		case "session_watch_upsert":
			s.upsertCount++
		}
	}
	return s
}

func renderWatchFamilySummary(summary watchFamilySummary) string {
	return fmt.Sprintf(
		"  %s\n    totals events=%d snapshot=%d upsert=%d terminal=%d errors=%d\n    last_event=%s subject=%s status=%s",
		infoBadge(sanitizeUIText(summary.family)),
		summary.eventCount,
		summary.snapshotCount,
		summary.upsertCount,
		summary.terminalCount,
		summary.errorCount,
		valueOrNA(sanitizeUIText(summary.lastEventType)),
		valueOrNA(sanitizeUIText(summary.lastSubject)),
		valueOrNA(sanitizeUIText(summary.lastStatus)),
	)
}
