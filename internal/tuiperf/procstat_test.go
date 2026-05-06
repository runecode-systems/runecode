package tuiperf

import "testing"

func TestParseProcStatLine(t *testing.T) {
	line := "1234 (runecode-tui) S 1 2 3 4 5 6 7 8 9 10 120 30 0 0 20 0 1 0 999 0 0 0"
	stat, err := ParseProcStatLine(line)
	if err != nil {
		t.Fatalf("ParseProcStatLine returned error: %v", err)
	}
	if stat.PID != 1234 {
		t.Fatalf("PID = %d, want 1234", stat.PID)
	}
	if stat.Comm != "runecode-tui" {
		t.Fatalf("Comm = %q, want runecode-tui", stat.Comm)
	}
	if stat.UserTicks != 120 || stat.SystemTicks != 30 {
		t.Fatalf("ticks = (%d,%d), want (120,30)", stat.UserTicks, stat.SystemTicks)
	}
	if stat.StartTicks != 999 {
		t.Fatalf("StartTicks = %d, want 999", stat.StartTicks)
	}
}

func TestParseProcStatLineRejectsInvalid(t *testing.T) {
	if _, err := ParseProcStatLine("bad"); err == nil {
		t.Fatal("ParseProcStatLine error = nil, want error")
	}
}
