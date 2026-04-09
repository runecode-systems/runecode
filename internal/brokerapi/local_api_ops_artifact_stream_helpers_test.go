package brokerapi

import "testing"

func assertStreamSeqMonotonic(t *testing.T, events []ArtifactStreamEvent) {
	t.Helper()
	for i := 1; i < len(events); i++ {
		if events[i].Seq <= events[i-1].Seq {
			t.Fatalf("stream seq not monotonic: prev=%d curr=%d", events[i-1].Seq, events[i].Seq)
		}
	}
}
