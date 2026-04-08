package artifacts

import "testing"

func TestReservedDataClassesFailClosedByDefault(t *testing.T) {
	store := newTestStore(t)
	_, err := store.Put(PutRequest{Payload: []byte("web"), ContentType: "text/plain", DataClass: DataClassWebQuery, ProvenanceReceiptHash: testDigest("b"), CreatedByRole: "workspace"})
	if err != ErrReservedDataClassDisabled {
		t.Fatalf("reserved class Put error = %v, want %v", err, ErrReservedDataClassDisabled)
	}
}
