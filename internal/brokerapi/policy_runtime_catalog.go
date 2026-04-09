package brokerapi

import (
	"sort"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type trustedImportRef struct {
	kind       string
	provenance string
	seq        int64
}

func (r policyRuntime) trustedPolicyCatalog() (trustedPolicyCatalog, error) {
	events, err := r.service.ReadAuditEvents()
	if err != nil {
		return trustedPolicyCatalog{}, err
	}
	imports := trustedImportsByDigest(events)
	catalog := trustedPolicyCatalog{byKind: map[string][]artifacts.ArtifactRecord{}, verifiers: []trustpolicy.VerifierRecord{}}
	for _, record := range r.service.List() {
		if imp, ok := imports[record.Reference.Digest]; ok && record.Reference.ProvenanceReceiptHash == imp.provenance {
			catalog.byKind[imp.kind] = append(catalog.byKind[imp.kind], record)
		}
		if !artifacts.IsTrustedVerifierArtifact(record, events) {
			continue
		}
		verifier, decodeErr := r.service.loadVerifierRecord(record)
		if decodeErr != nil {
			return trustedPolicyCatalog{}, decodeErr
		}
		catalog.verifiers = append(catalog.verifiers, verifier)
	}
	for kind := range catalog.byKind {
		sortByNewestFirst(catalog.byKind[kind])
	}
	return catalog, nil
}

func trustedImportsByDigest(events []artifacts.AuditEvent) map[string]trustedImportRef {
	out := map[string]trustedImportRef{}
	allowedKinds := allowedTrustedImportKinds()
	for _, event := range events {
		kind, digest, provenance, ok := trustedImportFromEvent(event, allowedKinds)
		if !ok {
			continue
		}
		if existing, seen := out[digest]; seen && existing.seq >= event.Seq {
			continue
		}
		out[digest] = trustedImportRef{kind: kind, provenance: provenance, seq: event.Seq}
	}
	return out
}

func allowedTrustedImportKinds() map[string]struct{} {
	return map[string]struct{}{
		artifacts.TrustedContractImportKindRoleManifest:    {},
		artifacts.TrustedContractImportKindRunCapability:   {},
		artifacts.TrustedContractImportKindStageCapability: {},
		artifacts.TrustedContractImportKindPolicyAllowlist: {},
		artifacts.TrustedContractImportKindPolicyRuleSet:   {},
	}
}

func trustedImportFromEvent(event artifacts.AuditEvent, allowedKinds map[string]struct{}) (string, string, string, bool) {
	if event.Type != artifacts.TrustedContractImportAuditEventType || event.Actor != "brokerapi" {
		return "", "", "", false
	}
	kind, _ := event.Details[artifacts.TrustedContractImportKindDetailKey].(string)
	if _, ok := allowedKinds[kind]; !ok {
		return "", "", "", false
	}
	digest, _ := event.Details[artifacts.TrustedContractImportArtifactDigestDetailKey].(string)
	provenance, _ := event.Details[artifacts.TrustedContractImportProvenanceDetailKey].(string)
	if !isSHA256Digest(digest) || !isSHA256Digest(provenance) {
		return "", "", "", false
	}
	return kind, digest, provenance, true
}

func sortByNewestFirst(records []artifacts.ArtifactRecord) {
	sort.Slice(records, func(i, j int) bool {
		left := records[i]
		right := records[j]
		if left.CreatedAt.Equal(right.CreatedAt) {
			return left.Reference.Digest > right.Reference.Digest
		}
		return left.CreatedAt.After(right.CreatedAt)
	})
}
