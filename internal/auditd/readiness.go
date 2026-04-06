package auditd

import "github.com/runecode-ai/runecode/internal/trustpolicy"

func (l *Ledger) Readiness() (trustpolicy.AuditdReadiness, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	state, err := l.recoverAndPersistStateLocked()
	if err != nil {
		return trustpolicy.AuditdReadiness{}, err
	}
	open, err := l.loadSegment(state.CurrentOpenSegmentID)
	if err != nil {
		return trustpolicy.AuditdReadiness{}, err
	}
	indexed, total, err := l.indexStatusLocked()
	if err != nil {
		return trustpolicy.AuditdReadiness{}, err
	}
	readiness := trustpolicy.AuditdReadiness{LocalOnly: true, ConsumptionChannel: "broker_local_api", RecoveryComplete: state.RecoveryComplete, AppendPositionStable: state.OpenFrameCount == len(open.Frames), CurrentSegmentWritable: open.Header.SegmentState == trustpolicy.AuditSegmentStateOpen, VerifierMaterialAvailable: hasVerificationInputs(l), DerivedIndexCaughtUp: indexed == total}
	readiness.Ready = readiness.RecoveryComplete && readiness.AppendPositionStable && readiness.CurrentSegmentWritable && readiness.VerifierMaterialAvailable && readiness.DerivedIndexCaughtUp
	if err := trustpolicy.ValidateAuditdReadinessContract(readiness); err != nil {
		return trustpolicy.AuditdReadiness{}, err
	}
	return readiness, nil
}
