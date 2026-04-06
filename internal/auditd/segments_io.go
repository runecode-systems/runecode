package auditd

import (
	"encoding/base64"
	"fmt"
	"path/filepath"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) loadSegment(segmentID string) (trustpolicy.AuditSegmentFilePayload, error) {
	segment := trustpolicy.AuditSegmentFilePayload{}
	if err := readJSONFile(filepath.Join(l.rootDir, segmentsDirName, segmentID+".json"), &segment); err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, err
	}
	if segment.Header.SegmentID != segmentID {
		return trustpolicy.AuditSegmentFilePayload{}, fmt.Errorf("segment file mismatch for %q", segmentID)
	}
	if segment.Header.Format != "audit_segment_framed_v1" || segment.Header.Writer != "auditd" {
		return trustpolicy.AuditSegmentFilePayload{}, fmt.Errorf("invalid segment header contract")
	}
	if segment.Header.SegmentState != segment.LifecycleMarker.State {
		return trustpolicy.AuditSegmentFilePayload{}, fmt.Errorf("segment header/lifecycle marker state mismatch")
	}
	if segment.Header.SegmentState == trustpolicy.AuditSegmentStateOpen && segment.TrailingPartialFrameBytes > 0 {
		return trustpolicy.AuditSegmentFilePayload{}, fmt.Errorf("trailing_partial_frame_bytes not supported in v0 runtime")
	}
	if segment.Header.SegmentState != trustpolicy.AuditSegmentStateOpen && len(segment.Frames) == 0 {
		return trustpolicy.AuditSegmentFilePayload{}, fmt.Errorf("non-open segment must have frames")
	}
	if segment.TrailingPartialFrameBytes > 0 {
		return trustpolicy.AuditSegmentFilePayload{}, fmt.Errorf("immutable segment cannot include trailing_partial_frame_bytes")
	}
	return segment, nil
}

func (l *Ledger) saveSegment(segment trustpolicy.AuditSegmentFilePayload) error {
	return writeCanonicalJSONFile(filepath.Join(l.rootDir, segmentsDirName, segment.Header.SegmentID+".json"), segment)
}

func (l *Ledger) rawSegmentFramedBytes(segment trustpolicy.AuditSegmentFilePayload) ([]byte, error) {
	raw := make([]byte, 0, len(segment.Frames)*256)
	for idx, frame := range segment.Frames {
		envelopeBytes, err := base64.StdEncoding.DecodeString(frame.CanonicalSignedEnvelopeBytes)
		if err != nil {
			return nil, fmt.Errorf("frame %d decode: %w", idx, err)
		}
		if int64(len(envelopeBytes)) != frame.ByteLength {
			return nil, fmt.Errorf("frame %d byte_length mismatch", idx)
		}
		raw = append(raw, envelopeBytes...)
		raw = append(raw, '\n')
	}
	return raw, nil
}
