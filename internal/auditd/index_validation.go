package auditd

import (
	"fmt"
	"strconv"
	"strings"
)

func validateDerivedIndexStructure(index derivedIndex) error {
	if index.TotalRecords < 0 {
		return fmt.Errorf("audit evidence index total_records must be non-negative")
	}
	if err := validateRecordDigestLookup(index.RecordDigestLookup); err != nil {
		return err
	}
	if err := validateSegmentSealLookup(index.SegmentSealLookup, index.SealChainIndexLookup); err != nil {
		return err
	}
	if err := validateSealChainIndexLookup(index.SealChainIndexLookup); err != nil {
		return err
	}
	return validateLatestVerificationReportDigest(index.LatestVerificationReportDigest)
}

func validateRecordDigestLookup(lookups map[string]RecordLookup) error {
	for recordDigest, lookup := range lookups {
		if err := validateRecordDigestLookupEntry(recordDigest, lookup); err != nil {
			return err
		}
	}
	return nil
}

func validateRecordDigestLookupEntry(recordDigest string, lookup RecordLookup) error {
	if err := validateRecordDigestIdentity(strings.TrimSpace(recordDigest)); err != nil {
		return fmt.Errorf("audit evidence index record_digest_lookup key %q invalid: %w", recordDigest, err)
	}
	if strings.TrimSpace(lookup.SegmentID) == "" {
		return fmt.Errorf("audit evidence index record_digest_lookup[%q].segment_id required", recordDigest)
	}
	if lookup.FrameIndex < 0 {
		return fmt.Errorf("audit evidence index record_digest_lookup[%q].frame_index must be non-negative", recordDigest)
	}
	return nil
}

func validateSegmentSealLookup(lookups map[string]SegmentSealLookup, chainLookup map[string]string) error {
	for segmentID, lookup := range lookups {
		if err := validateSegmentSealLookupEntry(segmentID, lookup, chainLookup); err != nil {
			return err
		}
	}
	return nil
}

func validateSegmentSealLookupEntry(segmentID string, lookup SegmentSealLookup, chainLookup map[string]string) error {
	if strings.TrimSpace(segmentID) == "" {
		return fmt.Errorf("audit evidence index segment_seal_lookup key required")
	}
	if _, err := digestFromIdentity(strings.TrimSpace(lookup.SealDigest)); err != nil {
		return fmt.Errorf("audit evidence index segment_seal_lookup[%q].seal_digest invalid: %w", segmentID, err)
	}
	if lookup.SealChainIndex < 0 {
		return fmt.Errorf("audit evidence index segment_seal_lookup[%q].seal_chain_index must be non-negative", segmentID)
	}
	chainKey := strconv.FormatInt(lookup.SealChainIndex, 10)
	if digest, exists := chainLookup[chainKey]; exists && strings.TrimSpace(digest) != strings.TrimSpace(lookup.SealDigest) {
		return fmt.Errorf("audit evidence index chain lookup mismatch for segment %q", segmentID)
	}
	return nil
}

func validateSealChainIndexLookup(chainLookup map[string]string) error {
	for chainIndex, sealDigest := range chainLookup {
		if err := validateSealChainIndexLookupEntry(chainIndex, sealDigest); err != nil {
			return err
		}
	}
	return nil
}

func validateSealChainIndexLookupEntry(chainIndex string, sealDigest string) error {
	if _, err := strconv.ParseInt(strings.TrimSpace(chainIndex), 10, 64); err != nil {
		return fmt.Errorf("audit evidence index seal_chain_index_lookup key %q invalid: %w", chainIndex, err)
	}
	if _, err := digestFromIdentity(strings.TrimSpace(sealDigest)); err != nil {
		return fmt.Errorf("audit evidence index seal_chain_index_lookup[%q] invalid: %w", chainIndex, err)
	}
	return nil
}

func validateLatestVerificationReportDigest(identity string) error {
	if digest := strings.TrimSpace(identity); digest != "" {
		if _, err := digestFromIdentity(digest); err != nil {
			return fmt.Errorf("audit evidence index latest_verification_report_digest invalid: %w", err)
		}
	}
	return nil
}
