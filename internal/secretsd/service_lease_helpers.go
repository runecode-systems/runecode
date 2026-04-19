package secretsd

import "time"

func (s *Service) accessDenied() error {
	s.state.Metrics.LeaseDenyCount++
	_ = s.persistState()
	return ErrAccessDenied
}

func deniedDeliveryKind(kind string) bool {
	return kind == deliveryKindEnvironmentVariable || kind == deliveryKindCLIArgument
}

func (s *Service) newLeaseRecord(req IssueLeaseRequest, deliveryKind string) (leaseRecord, error) {
	now := s.now().UTC()
	leaseID, err := randomLeaseID(s.rand)
	if err != nil {
		return leaseRecord{}, err
	}
	return leaseRecord{
		LeaseID:      leaseID,
		SecretRef:    req.SecretRef,
		ConsumerID:   req.ConsumerID,
		RoleKind:     req.RoleKind,
		Scope:        req.Scope,
		DeliveryKind: deliveryKind,
		GitBinding:   cloneGitLeaseBinding(req.GitBinding),
		IssuedAt:     now,
		ExpiresAt:    now.Add(time.Duration(effectiveTTL(req.TTLSeconds)) * time.Second),
		Status:       leaseStatusActive,
	}, nil
}
