package brokerapi

import (
	"strings"
	"sync"
)

const defaultGitProvider = "github"

type gitSetupState struct {
	mu               sync.RWMutex
	providerAccounts map[string]GitProviderAccountState
	identityProfiles map[string][]GitCommitIdentityProfile
	authPostures     map[string]GitAuthPostureState
	controlPlane     map[string]GitControlPlaneState
}

func newGitSetupState() *gitSetupState {
	s := &gitSetupState{
		providerAccounts: map[string]GitProviderAccountState{},
		identityProfiles: map[string][]GitCommitIdentityProfile{},
		authPostures:     map[string]GitAuthPostureState{},
		controlPlane:     map[string]GitControlPlaneState{},
	}
	s.seedProvider(defaultGitProvider)
	return s
}

func normalizeGitProvider(provider string) string {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "" {
		return defaultGitProvider
	}
	return p
}

func (s *gitSetupState) seedProvider(provider string) {
	p := normalizeGitProvider(provider)
	defaultProfile := GitCommitIdentityProfile{
		SchemaID:       "runecode.protocol.v0.GitCommitIdentityProfile",
		SchemaVersion:  "0.1.0",
		ProfileID:      "default",
		DisplayName:    "Default identity",
		AuthorName:     "RuneCode Operator",
		AuthorEmail:    "operator@example.invalid",
		CommitterName:  "RuneCode Operator",
		CommitterEmail: "operator@example.invalid",
		SignoffName:    "RuneCode Operator",
		SignoffEmail:   "operator@example.invalid",
		DefaultProfile: true,
	}
	s.providerAccounts[p] = GitProviderAccountState{SchemaID: "runecode.protocol.v0.GitProviderAccountState", SchemaVersion: "0.1.0", Provider: p, AccountID: "not_linked", AccountUsername: "not_linked", Linked: false, Source: "restored_state"}
	s.identityProfiles[p] = []GitCommitIdentityProfile{defaultProfile}
	s.authPostures[p] = GitAuthPostureState{SchemaID: "runecode.protocol.v0.GitAuthPostureState", SchemaVersion: "0.1.0", Provider: p, AuthStatus: "not_linked", BootstrapMode: "browser", HeadlessBootstrapSupported: true, InteractiveTokenFallbackSupport: true}
	s.controlPlane[p] = GitControlPlaneState{SchemaID: "runecode.protocol.v0.GitControlPlaneState", SchemaVersion: "0.1.0", Provider: p, DefaultIdentityProfileID: defaultProfile.ProfileID, LastSetupView: "overview", RecentRepositories: []string{}}
}

func (s *gitSetupState) snapshot(provider string) (GitProviderAccountState, []GitCommitIdentityProfile, GitAuthPostureState, GitControlPlaneState) {
	p := normalizeGitProvider(provider)
	s.mu.RLock()
	account, okAccount := s.providerAccounts[p]
	profiles, okProfiles := s.identityProfiles[p]
	auth, okAuth := s.authPostures[p]
	control, okControl := s.controlPlane[p]
	s.mu.RUnlock()
	if okAccount && okProfiles && okAuth && okControl {
		copied := append([]GitCommitIdentityProfile(nil), profiles...)
		return account, copied, auth, control
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.providerAccounts[p]; !ok {
		s.seedProvider(p)
	}
	account = s.providerAccounts[p]
	profiles = append([]GitCommitIdentityProfile(nil), s.identityProfiles[p]...)
	auth = s.authPostures[p]
	control = s.controlPlane[p]
	return account, profiles, auth, control
}

func (s *gitSetupState) upsertProfile(provider string, profile GitCommitIdentityProfile) (GitCommitIdentityProfile, GitControlPlaneState) {
	p := normalizeGitProvider(provider)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureProviderLocked(p)
	profiles := s.upsertIdentityProfilesLocked(p, profile)
	s.identityProfiles[p] = profiles
	control := s.updatedIdentityControlPlaneLocked(p, profile)
	s.controlPlane[p] = control
	return identityProfileByID(profiles, profile.ProfileID), control
}

func (s *gitSetupState) ensureProviderLocked(provider string) {
	if _, ok := s.providerAccounts[provider]; !ok {
		s.seedProvider(provider)
	}
}

func (s *gitSetupState) upsertIdentityProfilesLocked(provider string, profile GitCommitIdentityProfile) []GitCommitIdentityProfile {
	profiles := append([]GitCommitIdentityProfile(nil), s.identityProfiles[provider]...)
	profiles = replaceOrAppendIdentityProfile(profiles, profile)
	if profile.DefaultProfile {
		markDefaultIdentityProfile(profiles, profile.ProfileID)
	}
	return profiles
}

func replaceOrAppendIdentityProfile(profiles []GitCommitIdentityProfile, profile GitCommitIdentityProfile) []GitCommitIdentityProfile {
	for i := range profiles {
		if profiles[i].ProfileID == profile.ProfileID {
			profiles[i] = profile
			return profiles
		}
	}
	return append(profiles, profile)
}

func markDefaultIdentityProfile(profiles []GitCommitIdentityProfile, profileID string) {
	for i := range profiles {
		profiles[i].DefaultProfile = profiles[i].ProfileID == profileID
	}
}

func (s *gitSetupState) updatedIdentityControlPlaneLocked(provider string, profile GitCommitIdentityProfile) GitControlPlaneState {
	control := s.controlPlane[provider]
	control.LastSetupView = "identity"
	if profile.DefaultProfile || strings.TrimSpace(control.DefaultIdentityProfileID) == "" {
		control.DefaultIdentityProfileID = profile.ProfileID
	}
	return control
}

func identityProfileByID(profiles []GitCommitIdentityProfile, profileID string) GitCommitIdentityProfile {
	for i := range profiles {
		if profiles[i].ProfileID == profileID {
			return profiles[i]
		}
	}
	return GitCommitIdentityProfile{}
}

func (s *gitSetupState) applyAuthBootstrap(provider, mode string) (GitProviderAccountState, GitAuthPostureState, string, string, int, string) {
	p := normalizeGitProvider(provider)
	bootstrapMode := strings.TrimSpace(mode)
	if bootstrapMode == "" {
		bootstrapMode = "browser"
	}
	status := "pending"
	deviceURI := ""
	deviceCode := ""
	nextPoll := 0
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.providerAccounts[p]; !ok {
		s.seedProvider(p)
	}
	account := s.providerAccounts[p]
	auth := s.authPostures[p]
	auth.BootstrapMode = bootstrapMode
	if bootstrapMode == "device_code" {
		deviceURI = "https://github.com/login/device"
		deviceCode = "RUNE-CODE"
		nextPoll = 5
	} else {
		deviceURI = ""
		deviceCode = ""
	}
	account.Linked = false
	account.AccountID = "pending"
	account.AccountUsername = "pending"
	account.Source = "auth_bootstrap"
	auth.AuthStatus = "not_linked"
	s.providerAccounts[p] = account
	s.authPostures[p] = auth
	control := s.controlPlane[p]
	control.LastSetupView = "auth"
	s.controlPlane[p] = control
	return account, auth, status, deviceURI, nextPoll, deviceCode
}
