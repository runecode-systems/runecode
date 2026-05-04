package secretsd

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Open(root string) (*Service, error) {
	cleanRoot := strings.TrimSpace(root)
	if cleanRoot == "" {
		return nil, fmt.Errorf("state root is required")
	}
	release := lockServiceRoot(cleanRoot)
	defer release()
	if err := os.MkdirAll(filepath.Join(cleanRoot, secretsDirName), 0o700); err != nil {
		return nil, err
	}
	s := &Service{root: cleanRoot, now: time.Now, rand: rand.Reader}
	if err := s.loadState(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) ImportSecret(secretRef string, r io.Reader) (SecretMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(secretRef) == "" {
		return SecretMetadata{}, fmt.Errorf("secret_ref is required")
	}
	if r == nil {
		return SecretMetadata{}, fmt.Errorf("secret input is required")
	}
	material, err := io.ReadAll(r)
	if err != nil {
		return SecretMetadata{}, err
	}
	if len(material) == 0 {
		return SecretMetadata{}, fmt.Errorf("secret input is empty")
	}
	now := s.now().UTC()
	secretID, err := randomSecretID(s.rand)
	if err != nil {
		return SecretMetadata{}, err
	}
	digest := digestHex(material)
	materialFile := secretID + ".bin"
	if err := writeFileAtomic(filepath.Join(s.root, secretsDirName, materialFile), material, 0o600); err != nil {
		return SecretMetadata{}, err
	}
	s.state.Secrets[secretRef] = secretRecord{
		SecretID:       secretID,
		MaterialFile:   materialFile,
		MaterialDigest: digest,
		ImportedAt:     now,
		UpdatedAt:      now,
	}
	if err := s.persistState(); err != nil {
		return SecretMetadata{}, err
	}
	return SecretMetadata{SecretRef: secretRef, SecretID: secretID, MaterialDigest: digest, ImportedAt: now}, nil
}

func (s *Service) IssueLease(req IssueLeaseRequest) (Lease, error) {
	s.mu.Lock()
	deliveryKind, err := validateIssueLeaseRequest(req)
	if err != nil {
		s.mu.Unlock()
		return Lease{}, err
	}
	if _, ok := s.state.Secrets[req.SecretRef]; !ok {
		s.state.Metrics.LeaseDenyCount++
		_ = s.persistState()
		s.mu.Unlock()
		return Lease{}, ErrAccessDenied
	}
	rec, err := s.newLeaseRecord(req, deliveryKind)
	if err != nil {
		s.mu.Unlock()
		return Lease{}, err
	}
	s.state.Leases[rec.LeaseID] = rec
	s.state.Metrics.LeaseIssueCount++
	if err := s.persistState(); err != nil {
		s.mu.Unlock()
		return Lease{}, err
	}
	lease := rec.public()
	s.mu.Unlock()
	s.emitLeaseAuditEventLocked("issued", lease)
	return lease, nil
}

func (s *Service) RenewLease(req RenewLeaseRequest) (Lease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateBinding("placeholder", req.ConsumerID, req.RoleKind, req.Scope); err != nil {
		return Lease{}, err
	}
	lease, ok := s.state.Leases[req.LeaseID]
	if !ok {
		s.state.Metrics.LeaseDenyCount++
		_ = s.persistState()
		return Lease{}, ErrAccessDenied
	}
	if !lease.bindingMatches(req.ConsumerID, req.RoleKind, req.Scope) || lease.Status != leaseStatusActive || !lease.ExpiresAt.After(s.now().UTC()) {
		s.state.Metrics.LeaseDenyCount++
		_ = s.persistState()
		return Lease{}, ErrAccessDenied
	}
	now := s.now().UTC()
	lease.ExpiresAt = now.Add(time.Duration(effectiveTTL(req.TTLSeconds)) * time.Second)
	s.state.Leases[req.LeaseID] = lease
	s.state.Metrics.LeaseRenewCount++
	if err := s.persistState(); err != nil {
		return Lease{}, err
	}
	return lease.public(), nil
}

func (s *Service) RevokeLease(req RevokeLeaseRequest) (Lease, error) {
	s.mu.Lock()
	if err := validateBinding("placeholder", req.ConsumerID, req.RoleKind, req.Scope); err != nil {
		s.mu.Unlock()
		return Lease{}, err
	}
	lease, ok := s.state.Leases[req.LeaseID]
	if !ok {
		s.state.Metrics.LeaseDenyCount++
		_ = s.persistState()
		s.mu.Unlock()
		return Lease{}, ErrAccessDenied
	}
	if !lease.bindingMatches(req.ConsumerID, req.RoleKind, req.Scope) {
		s.state.Metrics.LeaseDenyCount++
		_ = s.persistState()
		s.mu.Unlock()
		return Lease{}, ErrAccessDenied
	}
	now := s.now().UTC()
	if lease.Status != leaseStatusRevoked {
		lease.Status = leaseStatusRevoked
		lease.RevokedAt = &now
		lease.Reason = strings.TrimSpace(req.Reason)
		s.state.Metrics.LeaseRevokeCount++
	}
	s.state.Leases[req.LeaseID] = lease
	if err := s.persistState(); err != nil {
		s.mu.Unlock()
		return Lease{}, err
	}
	public := lease.public()
	s.mu.Unlock()
	s.emitLeaseAuditEventLocked("revoked", public)
	return public, nil
}

func (s *Service) Retrieve(req RetrieveRequest) ([]byte, Lease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateBinding("placeholder", req.ConsumerID, req.RoleKind, req.Scope); err != nil {
		return nil, Lease{}, err
	}
	lease, err := s.retrievableLease(req)
	if err != nil {
		return nil, Lease{}, err
	}
	secretRec, ok := s.state.Secrets[lease.SecretRef]
	if !ok {
		return nil, Lease{}, fmt.Errorf("%w", ErrStateRecoveryFailed)
	}
	materialPath, err := s.secretMaterialPath(secretRec)
	if err != nil {
		return nil, Lease{}, err
	}
	material, err := os.ReadFile(materialPath)
	if err != nil {
		return nil, Lease{}, err
	}
	if digestHex(material) != secretRec.MaterialDigest {
		return nil, Lease{}, fmt.Errorf("%w", ErrStateRecoveryFailed)
	}
	return material, lease.public(), nil
}

func (s *Service) RevokeGitLeases(req RevokeGitLeasesRequest) ([]Lease, error) {
	s.mu.Lock()
	repositoryIdentity, actionRequestHash, policyContextHash, err := validateGitLeaseRevocationRequest(req)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	now := s.now().UTC()
	updated := make([]Lease, 0)
	for leaseID, lease := range s.state.Leases {
		if !gitLeaseRecordMatchesRevocation(lease, repositoryIdentity, actionRequestHash, policyContextHash) {
			continue
		}
		lease.Status = leaseStatusRevoked
		lease.RevokedAt = &now
		lease.Reason = strings.TrimSpace(req.Reason)
		s.state.Leases[leaseID] = lease
		s.state.Metrics.LeaseRevokeCount++
		updated = append(updated, lease.public())
	}
	if len(updated) == 0 {
		s.mu.Unlock()
		return []Lease{}, nil
	}
	if err := s.persistState(); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.mu.Unlock()
	for i := range updated {
		s.emitLeaseAuditEventLocked("revoked", updated[i])
	}
	return updated, nil
}
func (s *Service) retrievableLease(req RetrieveRequest) (leaseRecord, error) {
	requestedDeliveryKind := normalizeDeliveryKind(req.DeliveryKind)
	if deniedDeliveryKind(requestedDeliveryKind) {
		return leaseRecord{}, s.accessDenied()
	}
	lease, ok := s.state.Leases[req.LeaseID]
	if !ok || !lease.bindingMatches(req.ConsumerID, req.RoleKind, req.Scope) || lease.Status != leaseStatusActive || !lease.ExpiresAt.After(s.now().UTC()) {
		return leaseRecord{}, s.accessDenied()
	}
	if deniedDeliveryKind(lease.DeliveryKind) {
		return leaseRecord{}, s.accessDenied()
	}
	if requestedDeliveryKind != "" && requestedDeliveryKind != lease.DeliveryKind {
		return leaseRecord{}, s.accessDenied()
	}
	if lease.DeliveryKind == deliveryKindGitGateway && !gitLeaseBindingMatches(lease.GitBinding, req.GitUseContext) {
		return leaseRecord{}, s.accessDenied()
	}
	return lease, nil
}
