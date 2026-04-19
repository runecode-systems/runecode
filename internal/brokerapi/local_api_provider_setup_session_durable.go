package brokerapi

import "github.com/runecode-ai/runecode/internal/artifacts"

func providerSetupSessionToDurable(session ProviderSetupSession) artifacts.ProviderSetupSessionDurableState {
	return artifacts.ProviderSetupSessionDurableState{
		SchemaID:            session.SchemaID,
		SchemaVersion:       session.SchemaVersion,
		SetupSessionID:      session.SetupSessionID,
		ProviderProfileID:   session.ProviderProfileID,
		ProviderFamily:      session.ProviderFamily,
		SupportedAuthModes:  append([]string{}, session.SupportedAuthModes...),
		CurrentPhase:        session.CurrentPhase,
		CurrentAuthMode:     session.CurrentAuthMode,
		ValidationStatus:    session.ValidationStatus,
		ValidationAttemptID: session.ValidationAttemptID,
		ReadinessCommitted:  session.ReadinessCommitted,
		SecretIngressReady:  session.SecretIngressReady,
		CreatedAt:           session.CreatedAt,
		UpdatedAt:           session.UpdatedAt,
	}
}

func providerSetupSessionFromDurable(session artifacts.ProviderSetupSessionDurableState) ProviderSetupSession {
	return ProviderSetupSession{
		SchemaID:            session.SchemaID,
		SchemaVersion:       session.SchemaVersion,
		SetupSessionID:      session.SetupSessionID,
		ProviderProfileID:   session.ProviderProfileID,
		ProviderFamily:      session.ProviderFamily,
		SupportedAuthModes:  append([]string{}, session.SupportedAuthModes...),
		CurrentPhase:        session.CurrentPhase,
		CurrentAuthMode:     session.CurrentAuthMode,
		ValidationStatus:    session.ValidationStatus,
		ValidationAttemptID: session.ValidationAttemptID,
		ReadinessCommitted:  session.ReadinessCommitted,
		SecretIngressReady:  session.SecretIngressReady,
		CreatedAt:           session.CreatedAt,
		UpdatedAt:           session.UpdatedAt,
	}
}
