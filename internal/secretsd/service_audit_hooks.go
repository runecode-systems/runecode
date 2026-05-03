package secretsd

func (s *Service) SetLeaseAuditHookForTrustedRuntime(hook func(LeaseAuditEvent)) {
	if s == nil {
		return
	}
	if hook == nil {
		s.leaseAuditHook.Store(nil)
		return
	}
	s.leaseAuditHook.Store(&leaseAuditHookHolder{hook: hook})
}

func (s *Service) emitLeaseAuditEventLocked(action string, lease Lease) {
	if s == nil {
		return
	}
	hook := s.leaseAuditHookFunc()
	if hook == nil {
		return
	}
	hook(LeaseAuditEvent{Action: action, Lease: lease})
}

func (s *Service) leaseAuditHookFunc() func(LeaseAuditEvent) {
	if s == nil {
		return nil
	}
	raw := s.leaseAuditHook.Load()
	if raw == nil {
		return nil
	}
	return raw.hook
}
