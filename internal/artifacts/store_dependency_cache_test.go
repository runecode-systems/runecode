package artifacts

import "testing"

func TestDependencyCacheDataClassesRequireTrustedSource(t *testing.T) {
	store := newTestStore(t)
	_, err := store.Put(PutRequest{
		Payload:               []byte("unit"),
		ContentType:           "application/json",
		DataClass:             DataClassDependencyPayloadUnit,
		ProvenanceReceiptHash: testDigest("1"),
		CreatedByRole:         "dependency-fetch",
		TrustedSource:         false,
	})
	if err != ErrDependencyCacheTrustedSourceRequired {
		t.Fatalf("Put error = %v, want %v", err, ErrDependencyCacheTrustedSourceRequired)
	}
}

func TestDependencyCacheHitSemanticsExactAndFailClosed(t *testing.T) {
	store := newTestStore(t)
	requestDigest := testDigest("a")
	resolvedUnitDigest := testDigest("b")
	seedDependencyCacheRecord(t, store, testDigest("c"), requestDigest, resolvedUnitDigest)
	hit, err := dependencyCacheHitForTest(store, testDigest("c"), resolvedUnitDigest, requestDigest)
	if err != nil {
		t.Fatalf("DependencyCacheHit returned error: %v", err)
	}
	if !hit {
		t.Fatal("DependencyCacheHit = false, want true")
	}

	_ = putTrustedDependencyArtifactWithPayload(t, store, DataClassDependencyResolvedUnit, []byte(`{"kind":"unit-manifest-two"}`))
	store.state.DependencyCacheByRequest[requestDigest] = []string{resolvedUnitDigest, testDigest("f")}
	hit, err = dependencyCacheHitForTest(store, testDigest("c"), resolvedUnitDigest, requestDigest)
	if err != ErrDependencyCacheAmbiguousReuse {
		t.Fatalf("DependencyCacheHit ambiguous error = %v, want %v", err, ErrDependencyCacheAmbiguousReuse)
	}
	if hit {
		t.Fatal("DependencyCacheHit ambiguous = true, want false")
	}
}

func TestDependencyCacheRecordFailsClosedOnIncompleteState(t *testing.T) {
	store := newTestStore(t)
	batchManifest := putTrustedDependencyArtifact(t, store, DataClassDependencyBatchManifest, `{"kind":"batch-manifest"}`)
	unitManifest := putTrustedDependencyArtifact(t, store, DataClassDependencyResolvedUnit, `{"kind":"unit-manifest"}`)
	missingPayloadDigest := testDigest("f")
	err := store.RecordDependencyCacheBatch(DependencyCacheBatchRecord{
		BatchRequestDigest:  testDigest("0"),
		BatchManifestDigest: batchManifest.Digest,
		LockfileDigest:      testDigest("2"),
		RequestSetDigest:    testDigest("3"),
		ResolutionState:     "complete",
		CacheOutcome:        "miss_filled",
	}, []DependencyCacheResolvedUnitRecord{{
		ResolvedUnitDigest:   testDigest("1"),
		RequestDigest:        testDigest("4"),
		ManifestDigest:       unitManifest.Digest,
		PayloadDigest:        []string{missingPayloadDigest},
		IntegrityState:       "verified",
		MaterializationState: "derived_read_only",
	}})
	if err != ErrDependencyCacheIncompleteState {
		t.Fatalf("RecordDependencyCacheBatch error = %v, want %v", err, ErrDependencyCacheIncompleteState)
	}
}

func TestDependencyCacheResolvedUnitByRequest(t *testing.T) {
	store := newTestStore(t)
	batchManifest := putTrustedDependencyArtifact(t, store, DataClassDependencyBatchManifest, `{"kind":"batch-manifest"}`)
	unitManifest := putTrustedDependencyArtifact(t, store, DataClassDependencyResolvedUnit, `{"kind":"unit-manifest"}`)
	payload := putTrustedDependencyArtifact(t, store, DataClassDependencyPayloadUnit, `{"kind":"unit-payload"}`)

	requestDigest := testDigest("a")
	resolvedUnitDigest := testDigest("b")
	if err := store.RecordDependencyCacheBatch(DependencyCacheBatchRecord{
		BatchRequestDigest:  testDigest("c"),
		BatchManifestDigest: batchManifest.Digest,
		LockfileDigest:      testDigest("d"),
		RequestSetDigest:    testDigest("e"),
		ResolutionState:     "complete",
		CacheOutcome:        "miss_filled",
	}, []DependencyCacheResolvedUnitRecord{{
		ResolvedUnitDigest:   resolvedUnitDigest,
		RequestDigest:        requestDigest,
		ManifestDigest:       unitManifest.Digest,
		PayloadDigest:        []string{payload.Digest},
		IntegrityState:       "verified",
		MaterializationState: "derived_read_only",
	}}); err != nil {
		t.Fatalf("RecordDependencyCacheBatch returned error: %v", err)
	}

	unit, ok, err := store.DependencyCacheResolvedUnitByRequest(requestDigest)
	if err != nil {
		t.Fatalf("DependencyCacheResolvedUnitByRequest returned error: %v", err)
	}
	if !ok {
		t.Fatal("DependencyCacheResolvedUnitByRequest ok=false, want true")
	}
	if unit.ResolvedUnitDigest != resolvedUnitDigest {
		t.Fatalf("resolved_unit_digest = %q, want %q", unit.ResolvedUnitDigest, resolvedUnitDigest)
	}
}

func TestDependencyCacheHandoffByRequestUsesInternalArtifactFlow(t *testing.T) {
	store := newTestStore(t)
	requestDigest := testDigest("a")
	resolvedUnitDigest := testDigest("b")
	seedDependencyCacheRecord(t, store, testDigest("c"), requestDigest, resolvedUnitDigest)

	handoff, ok, err := store.DependencyCacheHandoffByRequest(DependencyCacheHandoffRequest{RequestDigest: requestDigest, ConsumerRole: "workspace"})
	if err != nil {
		t.Fatalf("DependencyCacheHandoffByRequest returned error: %v", err)
	}
	if !ok {
		t.Fatal("DependencyCacheHandoffByRequest ok=false, want true")
	}
	if handoff.HandoffMode != "broker_internal_artifact_handoff" {
		t.Fatalf("handoff_mode = %q, want broker_internal_artifact_handoff", handoff.HandoffMode)
	}
	if handoff.MaterializationMode != "derived_read_only" {
		t.Fatalf("materialization_mode = %q, want derived_read_only", handoff.MaterializationMode)
	}

	_, _, err = store.DependencyCacheHandoffByRequest(DependencyCacheHandoffRequest{RequestDigest: requestDigest, ConsumerRole: "model_gateway"})
	if err != ErrFlowDenied {
		t.Fatalf("DependencyCacheHandoffByRequest consumer error = %v, want %v", err, ErrFlowDenied)
	}
}

func seedDependencyCacheRecord(t *testing.T, store *Store, batchDigest, requestDigest, resolvedUnitDigest string) {
	t.Helper()
	batchManifest := putTrustedDependencyArtifact(t, store, DataClassDependencyBatchManifest, `{"kind":"batch-manifest"}`)
	unitManifest := putTrustedDependencyArtifact(t, store, DataClassDependencyResolvedUnit, `{"kind":"unit-manifest"}`)
	payload := putTrustedDependencyArtifact(t, store, DataClassDependencyPayloadUnit, `{"kind":"unit-payload"}`)
	if err := store.RecordDependencyCacheBatch(DependencyCacheBatchRecord{
		BatchRequestDigest:  batchDigest,
		BatchManifestDigest: batchManifest.Digest,
		LockfileDigest:      testDigest("d"),
		RequestSetDigest:    testDigest("e"),
		ResolutionState:     "complete",
		CacheOutcome:        "miss_filled",
	}, []DependencyCacheResolvedUnitRecord{{
		ResolvedUnitDigest:   resolvedUnitDigest,
		RequestDigest:        requestDigest,
		ManifestDigest:       unitManifest.Digest,
		PayloadDigest:        []string{payload.Digest},
		IntegrityState:       "verified",
		MaterializationState: "derived_read_only",
	}}); err != nil {
		t.Fatalf("RecordDependencyCacheBatch returned error: %v", err)
	}
}

func dependencyCacheHitForTest(store *Store, batchDigest, resolvedUnitDigest, requestDigest string) (bool, error) {
	return store.DependencyCacheHit(DependencyCacheHitRequest{
		BatchRequestDigest: batchDigest,
		ResolvedUnitDigest: resolvedUnitDigest,
		RequestDigest:      requestDigest,
	})
}

func TestSetPolicyRejectsDependencyCacheFailOpenConfiguration(t *testing.T) {
	store := newTestStore(t)
	policy := store.Policy()
	policy.DependencyCachePolicy.FailClosedOnIncompleteState = false
	err := store.SetPolicy(policy)
	if err == nil {
		t.Fatal("SetPolicy expected dependency cache fail-open rejection")
	}
}

func TestRecordDependencyCacheBatchSaveFailureRollsBackState(t *testing.T) {
	store := newTestStore(t)
	batchManifest := putTrustedDependencyArtifact(t, store, DataClassDependencyBatchManifest, `{"kind":"batch-manifest"}`)
	unitManifest := putTrustedDependencyArtifact(t, store, DataClassDependencyResolvedUnit, `{"kind":"unit-manifest"}`)
	payload := putTrustedDependencyArtifact(t, store, DataClassDependencyPayloadUnit, `{"kind":"unit-payload"}`)

	store.storeIO.statePath = store.rootDir
	err := store.RecordDependencyCacheBatch(DependencyCacheBatchRecord{
		BatchRequestDigest:  testDigest("c"),
		BatchManifestDigest: batchManifest.Digest,
		LockfileDigest:      testDigest("d"),
		RequestSetDigest:    testDigest("e"),
		ResolutionState:     "complete",
		CacheOutcome:        "miss_filled",
	}, []DependencyCacheResolvedUnitRecord{{
		ResolvedUnitDigest:   testDigest("b"),
		RequestDigest:        testDigest("a"),
		ManifestDigest:       unitManifest.Digest,
		PayloadDigest:        []string{payload.Digest},
		IntegrityState:       "verified",
		MaterializationState: "derived_read_only",
	}})
	if err == nil {
		t.Fatal("RecordDependencyCacheBatch expected save failure")
	}
	if len(store.state.DependencyCacheUnits) != 0 {
		t.Fatalf("dependency cache units mutated on save failure: %+v", store.state.DependencyCacheUnits)
	}
	if len(store.state.DependencyCacheByRequest) != 0 {
		t.Fatalf("dependency cache by-request mutated on save failure: %+v", store.state.DependencyCacheByRequest)
	}
	if len(store.state.DependencyCacheBatches) != 0 {
		t.Fatalf("dependency cache batches mutated on save failure: %+v", store.state.DependencyCacheBatches)
	}
}

func putTrustedDependencyArtifact(t *testing.T, store *Store, dataClass DataClass, payload string) ArtifactReference {
	t.Helper()
	return putTrustedDependencyArtifactWithPayload(t, store, dataClass, []byte(payload))
}

func putTrustedDependencyArtifactWithPayload(t *testing.T, store *Store, dataClass DataClass, payload []byte) ArtifactReference {
	t.Helper()
	ref, err := store.Put(PutRequest{
		Payload:               payload,
		ContentType:           "application/json",
		DataClass:             dataClass,
		ProvenanceReceiptHash: testDigest("9"),
		CreatedByRole:         "dependency-fetch",
		TrustedSource:         true,
	})
	if err != nil {
		t.Fatalf("Put trusted dependency artifact returned error: %v", err)
	}
	return ref
}
