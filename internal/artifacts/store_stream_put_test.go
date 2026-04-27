package artifacts

import (
	"io"
	"strings"
	"testing"
)

func TestPutStreamPersistsPayloadWithoutFullBufferingPath(t *testing.T) {
	store := newTestStore(t)
	payload := strings.Repeat("stream-payload-", 4096)
	ref, err := store.PutStream(PutStreamRequest{
		Reader:                strings.NewReader(payload),
		ContentType:           "application/octet-stream",
		DataClass:             DataClassDependencyPayloadUnit,
		ProvenanceReceiptHash: testDigest("1"),
		CreatedByRole:         "dependency-fetch",
		TrustedSource:         true,
	})
	if err != nil {
		t.Fatalf("PutStream returned error: %v", err)
	}
	r, err := store.Get(ref.Digest)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	b, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		t.Fatalf("ReadAll returned error: %v", readErr)
	}
	if string(b) != payload {
		t.Fatalf("stored payload mismatch")
	}
	if ref.SizeBytes != int64(len(payload)) {
		t.Fatalf("size_bytes = %d, want %d", ref.SizeBytes, len(payload))
	}
}

func TestPutStreamRejectsJSONCanonicalizationByDesign(t *testing.T) {
	store := newTestStore(t)
	_, err := store.PutStream(PutStreamRequest{
		Reader:                strings.NewReader(`{"a":1}`),
		ContentType:           "application/json",
		DataClass:             DataClassDependencyPayloadUnit,
		ProvenanceReceiptHash: testDigest("1"),
		CreatedByRole:         "dependency-fetch",
		TrustedSource:         true,
	})
	if err == nil {
		t.Fatal("PutStream expected json canonicalization rejection")
	}
}
