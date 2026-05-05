package tuiperf

import "testing"

func TestP95Millis(t *testing.T) {
	v, err := P95Millis([]float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100})
	if err != nil {
		t.Fatalf("P95Millis returned error: %v", err)
	}
	if v != 90 {
		t.Fatalf("p95 = %.2f, want 90", v)
	}
}

func TestP95MillisRejectsEmpty(t *testing.T) {
	if _, err := P95Millis(nil); err == nil {
		t.Fatal("P95Millis error = nil, want error")
	}
}
