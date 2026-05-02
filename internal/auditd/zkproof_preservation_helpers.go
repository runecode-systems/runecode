package auditd

import (
	"encoding/json"
	"os"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type auditProofBindingEvidenceSnapshot struct {
	protocolBundleManifestHashes         []string
	runtimeImageDescriptorDigests        []string
	attestationEvidenceDigests           []string
	appliedHardeningPostureDigests       []string
	sessionBindingDigests                []string
	projectSubstrateSnapshotDigests      []string
	attestationVerificationRecordDigests []string
}

type externalAnchorIdentityBindingSnapshot struct {
	typedRequestHashes     []string
	actionRequestHashes    []string
	policyDecisionHashes   []string
	requiredApprovalIDs    []string
	approvalRequestHashes  []string
	approvalDecisionHashes []string
}

type proofSidecarDigestSnapshot struct {
	segmentSeals       []string
	receipts           []string
	reports            []string
	externalEvidence   []string
	externalSidecars   []string
	proofBindings      []string
	proofArtifacts     []string
	proofVerifications []string
}

type proofVerificationContractSnapshot struct {
	verifierRecords []string
	eventCatalogs   []string
	signerEvidence  []string
	storagePosture  []string
}

func appendExternalAnchorIdentityBindingSnapshot(snapshot externalAnchorIdentityBindingSnapshot, rec trustpolicy.ExternalAnchorEvidencePayload) externalAnchorIdentityBindingSnapshot {
	snapshot.typedRequestHashes = appendOptionalDigestIdentity(snapshot.typedRequestHashes, rec.TypedRequestHash)
	snapshot.actionRequestHashes = appendOptionalDigestIdentity(snapshot.actionRequestHashes, rec.ActionRequestHash)
	snapshot.policyDecisionHashes = appendOptionalDigestIdentity(snapshot.policyDecisionHashes, rec.PolicyDecisionHash)
	snapshot.requiredApprovalIDs = appendIdentityUnique(snapshot.requiredApprovalIDs, strings.TrimSpace(rec.RequiredApprovalID))
	snapshot.approvalRequestHashes = appendOptionalDigestIdentity(snapshot.approvalRequestHashes, rec.ApprovalRequestHash)
	snapshot.approvalDecisionHashes = appendOptionalDigestIdentity(snapshot.approvalDecisionHashes, rec.ApprovalDecisionHash)
	return snapshot
}

func appendOptionalDigestIdentity(dst []string, digest *trustpolicy.Digest) []string {
	if digest == nil {
		return dst
	}
	return appendIdentityUnique(dst, mustDigestIdentityString(*digest))
}

func sortExternalAnchorIdentityBindingSnapshot(snapshot *externalAnchorIdentityBindingSnapshot) {
	sort.Strings(snapshot.typedRequestHashes)
	sort.Strings(snapshot.actionRequestHashes)
	sort.Strings(snapshot.policyDecisionHashes)
	sort.Strings(snapshot.requiredApprovalIDs)
	sort.Strings(snapshot.approvalRequestHashes)
	sort.Strings(snapshot.approvalDecisionHashes)
}

func canonicalDigestIdentityForJSONPath(path string) (string, error) {
	raw := json.RawMessage{}
	if err := readJSONFile(path, &raw); err != nil {
		return "", err
	}
	digest, err := canonicalDigest(raw)
	if err != nil {
		return "", err
	}
	return digest.Identity()
}

func canonicalDigestIdentityForOptionalJSONPath(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return canonicalDigestIdentityForJSONPath(path)
}

func wrapIdentityList(identity string) []string {
	if strings.TrimSpace(identity) == "" {
		return nil
	}
	return []string{identity}
}

func appendIdentityUnique(values []string, identity string) []string {
	trimmed := strings.TrimSpace(identity)
	if trimmed == "" {
		return values
	}
	for i := range values {
		if values[i] == trimmed {
			return values
		}
	}
	return append(values, trimmed)
}
