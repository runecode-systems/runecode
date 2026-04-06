package protocolschema

func validAuditEvent() map[string]any {
	return map[string]any{
		"schema_id":                       "runecode.protocol.v0.AuditEvent",
		"schema_version":                  "0.5.0",
		"audit_event_type":                "session_open",
		"emitter_stream_id":               "broker-stream-1",
		"seq":                             1,
		"occurred_at":                     "2026-03-13T12:15:00Z",
		"principal":                       manifestPrincipal(),
		"active_role_manifest_hash":       testDigestValue("9"),
		"active_capability_manifest_hash": testDigestValue("8"),
		"protocol_bundle_manifest_hash":   testDigestValue("7"),
		"event_payload_schema_id":         "runecode.protocol.audit.payload.session-open.v0",
		"event_payload":                   map[string]any{"session_id": "session-1"},
		"event_payload_hash":              testDigestValue("c"),
		"scope": map[string]any{
			"workspace_id": "workspace-1",
			"run_id":       "run-1",
		},
		"correlation": map[string]any{
			"session_id":   "session-1",
			"operation_id": "op-1",
		},
	}
}

func validGatewayAuditEvent() map[string]any {
	event := validAuditEvent()
	event["audit_event_type"] = "model_egress"
	event["previous_event_hash"] = testDigestValue("0")
	event["gateway_context"] = map[string]any{
		"egress_category":        "model",
		"allowlist_ref":          testDigestValue("d"),
		"destination_descriptor": "api.openai.com:443",
	}
	event["subject_ref"] = map[string]any{
		"object_family": "artifact_reference",
		"digest":        testDigestValue("7"),
		"ref_role":      "target",
	}
	event["cause_refs"] = []any{map[string]any{
		"object_family": "policy_decision",
		"digest":        testDigestValue("e"),
		"ref_role":      "policy_cause",
	}}
	event["related_refs"] = []any{map[string]any{
		"object_family": "audit_receipt",
		"digest":        testDigestValue("f"),
		"ref_role":      "evidence",
	}}
	event["signer_evidence_refs"] = []any{map[string]any{
		"object_family": "verifier_record",
		"digest":        testDigestValue("a"),
		"ref_role":      "admissibility",
	}}
	return event
}

func invalidAuditEventWithoutProtocolBundleManifestHash() map[string]any {
	event := validAuditEvent()
	delete(event, "protocol_bundle_manifest_hash")
	return event
}

func invalidAuditEventWithLegacySchemaBundleHash() map[string]any {
	event := validAuditEvent()
	event["schema_bundle_manifest_hash"] = testDigestValue("9")
	return event
}

func invalidAuditEventWithoutPayloadHash() map[string]any {
	event := validAuditEvent()
	delete(event, "event_payload_hash")
	return event
}

func invalidAuditEventWithBadType() map[string]any {
	event := validAuditEvent()
	event["audit_event_type"] = "model-egress"
	return event
}

func invalidAuditEventWithoutEmitterStreamID() map[string]any {
	event := validAuditEvent()
	delete(event, "emitter_stream_id")
	return event
}

func validAuditEventContractCatalog() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.AuditEventContractCatalog",
		"schema_version": "0.1.0",
		"catalog_id":     "audit_event_contract_v0",
		"entries": []any{
			map[string]any{
				"audit_event_type":                  "model_egress",
				"allowed_payload_schema_ids":        []any{"runecode.protocol.audit.payload.model-egress.v0"},
				"allowed_signer_purposes":           []any{"gateway_emitter"},
				"allowed_signer_scopes":             []any{"run", "stage"},
				"required_scope_fields":             []any{"workspace_id", "run_id", "stage_id"},
				"required_correlation_fields":       []any{"session_id", "operation_id"},
				"require_subject_ref":               true,
				"allowed_subject_ref_roles":         []any{"target"},
				"allowed_cause_ref_roles":           []any{"policy_cause"},
				"allowed_related_ref_roles":         []any{"evidence", "artifact"},
				"require_gateway_context":           true,
				"allowed_gateway_egress_categories": []any{"model"},
				"require_signer_evidence_refs":      false,
				"allowed_signer_evidence_ref_roles": []any{"admissibility", "binding"},
			},
		},
	}
}

func invalidAuditEventContractCatalogWithoutEntries() map[string]any {
	catalog := validAuditEventContractCatalog()
	delete(catalog, "entries")
	return catalog
}

func invalidAuditEventContractCatalogGatewayRule() map[string]any {
	catalog := validAuditEventContractCatalog()
	entry := catalog["entries"].([]any)[0].(map[string]any)
	entry["allowed_gateway_egress_categories"] = []any{}
	return catalog
}

func validAuditReceipt() map[string]any {
	return map[string]any{
		"schema_id":          "runecode.protocol.v0.AuditReceipt",
		"schema_version":     "0.4.0",
		"subject_digest":     testDigestValue("c"),
		"audit_receipt_kind": "anchor",
		"subject_family":     "audit_segment_seal",
		"recorder":           manifestPrincipal(),
		"recorded_at":        "2026-03-13T12:16:00Z",
	}
}

func validAuditReceiptWithPayload() map[string]any {
	receipt := validAuditReceipt()
	receipt["receipt_payload_schema_id"] = "runecode.protocol.audit.receipt.anchor.v0"
	receipt["receipt_payload"] = map[string]any{"anchor_kind": "local"}
	return receipt
}

func invalidAuditReceiptWithoutPayloadSchema() map[string]any {
	receipt := validAuditReceiptWithPayload()
	delete(receipt, "receipt_payload_schema_id")
	return receipt
}

func invalidAuditReceiptWithBadKind() map[string]any {
	receipt := validAuditReceipt()
	receipt["audit_receipt_kind"] = "Write-Ack"
	return receipt
}

func validImportAuditReceipt() map[string]any {
	receipt := validAuditReceipt()
	receipt["audit_receipt_kind"] = "import"
	receipt["receipt_payload_schema_id"] = "runecode.protocol.audit.receipt.import_restore_provenance.v0"
	receipt["receipt_payload"] = validImportRestoreReceiptPayload("import")
	return receipt
}

func validRestoreAuditReceipt() map[string]any {
	receipt := validAuditReceipt()
	receipt["audit_receipt_kind"] = "restore"
	receipt["receipt_payload_schema_id"] = "runecode.protocol.audit.receipt.import_restore_provenance.v0"
	receipt["receipt_payload"] = validImportRestoreReceiptPayload("restore")
	return receipt
}

func validImportRestoreReceiptPayload(action string) map[string]any {
	return map[string]any{
		"provenance_action":       action,
		"segment_file_hash_scope": "raw_framed_segment_bytes_v1",
		"imported_segments":       []any{validImportRestoreSegmentLink()},
		"source_manifest_digests": []any{testDigestValue("4"), testDigestValue("5")},
		"source_instance_id":      "instance-source-01",
		"operator":                manifestPrincipal(),
		"authority_context":       validImportRestoreAuthorityContext(),
	}
}

func validImportRestoreSegmentLink() map[string]any {
	return map[string]any{
		"imported_segment_seal_digest": testDigestValue("1"),
		"imported_segment_root":        testDigestValue("2"),
		"source_segment_file_hash":     testDigestValue("3"),
		"local_segment_file_hash":      testDigestValue("3"),
		"byte_identity_verified":       true,
	}
}

func validImportRestoreAuthorityContext() map[string]any {
	return map[string]any{
		"authority_kind":                "operator",
		"authority_id":                  "op-restore-1",
		"authorization_manifest_digest": testDigestValue("6"),
		"note":                          "approved by on-call authority",
	}
}

func invalidImportAuditReceiptWithWrongPayloadSchema() map[string]any {
	receipt := validImportAuditReceipt()
	receipt["receipt_payload_schema_id"] = "runecode.protocol.audit.receipt.anchor.v0"
	return receipt
}

func invalidImportAuditReceiptWithoutByteIdentity() map[string]any {
	receipt := validImportAuditReceipt()
	payload := receipt["receipt_payload"].(map[string]any)
	segments := payload["imported_segments"].([]any)
	segment := segments[0].(map[string]any)
	delete(segment, "byte_identity_verified")
	return receipt
}

func invalidRestoreAuditReceiptWithImportAction() map[string]any {
	receipt := validRestoreAuditReceipt()
	payload := receipt["receipt_payload"].(map[string]any)
	payload["provenance_action"] = "import"
	return receipt
}

func validAuditSegmentSeal() map[string]any {
	return map[string]any{
		"schema_id":                     "runecode.protocol.v0.AuditSegmentSeal",
		"schema_version":                "0.2.0",
		"segment_id":                    "segment-0001",
		"sealed_after_state":            "open",
		"segment_state":                 "sealed",
		"segment_cut":                   map[string]any{"ownership_scope": "instance_global", "max_segment_bytes": 1048576, "cut_trigger": "size_window"},
		"event_count":                   20,
		"first_record_digest":           testDigestValue("1"),
		"last_record_digest":            testDigestValue("2"),
		"merkle_profile":                "sha256_ordered_dse_v1",
		"merkle_root":                   testDigestValue("3"),
		"segment_file_hash_scope":       "raw_framed_segment_bytes_v1",
		"segment_file_hash":             testDigestValue("4"),
		"seal_chain_index":              0,
		"anchoring_subject":             "audit_segment_seal",
		"sealed_at":                     "2026-03-13T12:20:00Z",
		"protocol_bundle_manifest_hash": testDigestValue("5"),
	}
}

func validAuditSegmentSealWithPreviousSeal() map[string]any {
	seal := validAuditSegmentSeal()
	seal["segment_cut"] = map[string]any{"ownership_scope": "instance_global", "max_segment_duration_seconds": 900, "cut_trigger": "time_window"}
	seal["seal_chain_index"] = 1
	seal["previous_seal_digest"] = testDigestValue("6")
	seal["segment_state"] = "anchored"
	seal["seal_reason"] = "size_threshold"
	return seal
}

func invalidAuditSegmentSealWithPerRunCutOwnership() map[string]any {
	seal := validAuditSegmentSeal()
	segmentCut := seal["segment_cut"].(map[string]any)
	segmentCut["ownership_scope"] = "per_run"
	return seal
}

func invalidAuditSegmentSealWithoutPreviousAtNonGenesisIndex() map[string]any {
	seal := validAuditSegmentSeal()
	seal["seal_chain_index"] = 2
	return seal
}

func invalidAuditSegmentSealWithoutEventCount() map[string]any {
	seal := validAuditSegmentSeal()
	delete(seal, "event_count")
	return seal
}

func validAuditVerificationReport() map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.v0.AuditVerificationReport",
		"schema_version":           "0.1.0",
		"verified_at":              "2026-03-13T12:30:00Z",
		"verification_scope":       map[string]any{"scope_kind": "instance"},
		"cryptographically_valid":  true,
		"historically_admissible":  true,
		"currently_degraded":       true,
		"integrity_status":         "ok",
		"anchoring_status":         "degraded",
		"storage_posture_status":   "ok",
		"segment_lifecycle_status": "ok",
		"degraded_reasons":         []any{"anchor_receipt_missing"},
		"hard_failures":            []any{},
		"findings": []any{map[string]any{
			"code":      "anchor_receipt_missing",
			"dimension": "anchoring",
			"severity":  "warning",
			"message":   "No anchor receipts were present for one or more sealed segments.",
		}},
	}
}

func validAuditVerificationReportWithDigestFinding() map[string]any {
	report := validAuditVerificationReport()
	report["verification_scope"] = map[string]any{
		"scope_kind":      "segment",
		"last_segment_id": "segment-0001",
	}
	report["findings"] = []any{map[string]any{
		"code":                   "stream_sequence_gap",
		"dimension":              "integrity",
		"severity":               "error",
		"message":                "Emitter stream continuity check found a sequence gap.",
		"segment_id":             "segment-0001",
		"subject_record_digest":  testDigestValue("7"),
		"related_record_digests": []any{testDigestValue("8")},
	}}
	report["hard_failures"] = []any{"stream_sequence_gap"}
	report["degraded_reasons"] = []any{}
	report["currently_degraded"] = false
	report["cryptographically_valid"] = false
	return report
}

func invalidAuditVerificationReportWithBadSeverity() map[string]any {
	report := validAuditVerificationReport()
	report["findings"] = []any{map[string]any{
		"code":      "anchor_receipt_missing",
		"dimension": "anchoring",
		"severity":  "critical",
		"message":   "invalid severity",
	}}
	return report
}

func validOpenAuditSegmentFileWithTornTrailingFrame() map[string]any {
	return map[string]any{
		"schema_id":      "runecode.protocol.v0.AuditSegmentFile",
		"schema_version": "0.1.0",
		"header": map[string]any{
			"format":        "audit_segment_framed_v1",
			"segment_id":    "segment-0001",
			"segment_state": "open",
			"created_at":    "2026-03-13T12:10:00Z",
			"writer":        "auditd",
		},
		"frames": []any{map[string]any{
			"record_digest":                   testDigestValue("1"),
			"byte_length":                     123,
			"canonical_signed_envelope_bytes": "eyJzY2hlbWFfaWQiOiJydW5lY29kZS5wcm90b2NvbC52MC5TaWduZWRPYmplY3RFbnZlbG9wZSJ9",
		}},
		"lifecycle_marker": map[string]any{
			"state":     "open",
			"marked_at": "2026-03-13T12:10:05Z",
		},
		"trailing_partial_frame_bytes": 21,
	}
}

func invalidSealedAuditSegmentFileWithTrailingBytes() map[string]any {
	segment := validOpenAuditSegmentFileWithTornTrailingFrame()
	header := segment["header"].(map[string]any)
	header["segment_state"] = "sealed"
	marker := segment["lifecycle_marker"].(map[string]any)
	marker["state"] = "sealed"
	return segment
}

func validAnchoredAuditSegmentFile() map[string]any {
	segment := validOpenAuditSegmentFileWithTornTrailingFrame()
	delete(segment, "trailing_partial_frame_bytes")
	header := segment["header"].(map[string]any)
	header["segment_state"] = "anchored"
	marker := segment["lifecycle_marker"].(map[string]any)
	marker["state"] = "anchored"
	return segment
}

func validImportedAuditSegmentFile() map[string]any {
	segment := validOpenAuditSegmentFileWithTornTrailingFrame()
	delete(segment, "trailing_partial_frame_bytes")
	header := segment["header"].(map[string]any)
	header["segment_state"] = "imported"
	marker := segment["lifecycle_marker"].(map[string]any)
	marker["state"] = "imported"
	return segment
}

func invalidAuditSegmentFileWithoutFrameDigest() map[string]any {
	segment := validOpenAuditSegmentFileWithTornTrailingFrame()
	frames := segment["frames"].([]any)
	frame := frames[0].(map[string]any)
	delete(frame, "record_digest")
	return segment
}
