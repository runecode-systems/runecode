package brokerapi

import (
	"math"
	"sync"
)

type GatewayQuotaLimits struct {
	MaxRequestUnits     int64
	MaxInputTokens      int64
	MaxOutputTokens     int64
	MaxStreamedBytes    int64
	MaxConcurrencyUnits int64
	MaxSpendMicros      int64
	MaxEntitlementUnits int64
}

type gatewayQuotaBackend struct {
	mu     sync.Mutex
	limits GatewayQuotaLimits
	state  map[string]gatewayQuotaState
}

type gatewayQuotaState struct {
	RequestUnits     int64
	InputTokens      int64
	OutputTokens     int64
	StreamedBytes    int64
	ConcurrencyUnits int64
	SpendMicros      int64
	EntitlementUnits int64
}

func newGatewayQuotaBackend() *gatewayQuotaBackend {
	return &gatewayQuotaBackend{state: map[string]gatewayQuotaState{}}
}

func (q *gatewayQuotaBackend) setLimits(limits GatewayQuotaLimits) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.limits = limits
}

func (q *gatewayQuotaBackend) evaluateAndApply(key string, quota gatewayQuotaContextPayload) (string, map[string]any, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if reason, details, blocked := quotaMeterValidationReason(quota); blocked {
		return reason, details, true
	}

	current := q.state[key]
	next := current
	if reason, details, blocked := applyQuotaMeterDeltas(&next, quota.Meters); blocked {
		return reason, details, true
	}

	if reason, details, blocked := q.quotaLimitReason(next, quota); blocked {
		return reason, details, true
	}

	q.state[key] = next
	return "", nil, false
}

func (q *gatewayQuotaBackend) quotaLimitReason(next gatewayQuotaState, quota gatewayQuotaContextPayload) (string, map[string]any, bool) {
	reason, details, blocked := admissionQuotaLimitReason(q.limits, next)
	if blocked {
		return reason, details, true
	}
	if skipStreamQuotaEnforcement(quota) {
		return "", nil, false
	}
	return streamQuotaLimitReason(q.limits, next, quota)
}

func skipStreamQuotaEnforcement(quota gatewayQuotaContextPayload) bool {
	return quota.Phase == "stream" && !quota.EnforceDuringStream
}

func quotaExceededDetails(reason string, limit, observed int64) string {
	_ = limit
	_ = observed
	return reason
}

func exceedsQuotaLimit(limit, observed int64) bool {
	return limit > 0 && observed > limit
}

func quotaMeterValidationReason(quota gatewayQuotaContextPayload) (string, map[string]any, bool) {
	for meter, value := range map[string]*int64{
		"request_units":     quota.Meters.RequestUnits,
		"input_tokens":      quota.Meters.InputTokens,
		"output_tokens":     quota.Meters.OutputTokens,
		"streamed_bytes":    quota.Meters.StreamedBytes,
		"concurrency_units": quota.Meters.ConcurrencyUnits,
		"spend_micros":      quota.Meters.SpendMicros,
		"entitlement_units": quota.Meters.EntitlementUnits,
	} {
		if value != nil && *value < 0 {
			return "invalid_quota_meter_negative", map[string]any{"meter": meter, "value": *value}, true
		}
	}
	return "", nil, false
}

func safeAddInt64(current, delta int64) (int64, bool) {
	if delta > 0 && current > math.MaxInt64-delta {
		return 0, false
	}
	if delta < 0 && current < math.MinInt64-delta {
		return 0, false
	}
	return current + delta, true
}

func admissionQuotaLimitReason(limits GatewayQuotaLimits, next gatewayQuotaState) (string, map[string]any, bool) {
	checks := []struct {
		reason   string
		limit    int64
		observed int64
	}{
		{reason: "quota_admission_limit_exceeded_request_units", limit: limits.MaxRequestUnits, observed: next.RequestUnits},
		{reason: "quota_admission_limit_exceeded_input_tokens", limit: limits.MaxInputTokens, observed: next.InputTokens},
		{reason: "quota_admission_limit_exceeded_output_tokens", limit: limits.MaxOutputTokens, observed: next.OutputTokens},
		{reason: "quota_admission_limit_exceeded_concurrency_units", limit: limits.MaxConcurrencyUnits, observed: next.ConcurrencyUnits},
		{reason: "quota_admission_limit_exceeded_spend_micros", limit: limits.MaxSpendMicros, observed: next.SpendMicros},
		{reason: "quota_admission_limit_exceeded_entitlement_units", limit: limits.MaxEntitlementUnits, observed: next.EntitlementUnits},
	}
	for _, check := range checks {
		if !exceedsQuotaLimit(check.limit, check.observed) {
			continue
		}
		return quotaExceededDetails(check.reason, check.limit, check.observed), map[string]any{"limit": check.limit, "observed": check.observed}, true
	}
	return "", nil, false
}

func streamQuotaLimitReason(limits GatewayQuotaLimits, next gatewayQuotaState, quota gatewayQuotaContextPayload) (string, map[string]any, bool) {
	if exceedsQuotaLimit(limits.MaxStreamedBytes, next.StreamedBytes) {
		return quotaExceededDetails("quota_stream_limit_exceeded_streamed_bytes", limits.MaxStreamedBytes, next.StreamedBytes), map[string]any{"limit": limits.MaxStreamedBytes, "observed": next.StreamedBytes}, true
	}
	if quota.Phase == "stream" && quota.StreamLimitBytes != nil && next.StreamedBytes > *quota.StreamLimitBytes {
		return "quota_stream_limit_exceeded_streamed_bytes", map[string]any{"stream_limit_bytes": *quota.StreamLimitBytes, "streamed_bytes": next.StreamedBytes}, true
	}
	return "", nil, false
}

func applyQuotaMeterDeltas(next *gatewayQuotaState, meters gatewayQuotaMetersPayload) (string, map[string]any, bool) {
	updates := []struct {
		meter string
		delta *int64
		get   func(*gatewayQuotaState) *int64
	}{
		{meter: "request_units", delta: meters.RequestUnits, get: func(s *gatewayQuotaState) *int64 { return &s.RequestUnits }},
		{meter: "input_tokens", delta: meters.InputTokens, get: func(s *gatewayQuotaState) *int64 { return &s.InputTokens }},
		{meter: "output_tokens", delta: meters.OutputTokens, get: func(s *gatewayQuotaState) *int64 { return &s.OutputTokens }},
		{meter: "streamed_bytes", delta: meters.StreamedBytes, get: func(s *gatewayQuotaState) *int64 { return &s.StreamedBytes }},
		{meter: "concurrency_units", delta: meters.ConcurrencyUnits, get: func(s *gatewayQuotaState) *int64 { return &s.ConcurrencyUnits }},
		{meter: "spend_micros", delta: meters.SpendMicros, get: func(s *gatewayQuotaState) *int64 { return &s.SpendMicros }},
		{meter: "entitlement_units", delta: meters.EntitlementUnits, get: func(s *gatewayQuotaState) *int64 { return &s.EntitlementUnits }},
	}
	for _, update := range updates {
		if reason, details, blocked := applySingleQuotaMeterDelta(next, update.meter, update.delta, update.get); blocked {
			return reason, details, true
		}
	}
	return "", nil, false
}

func applySingleQuotaMeterDelta(next *gatewayQuotaState, meter string, delta *int64, getCurrent func(*gatewayQuotaState) *int64) (string, map[string]any, bool) {
	if delta == nil {
		return "", nil, false
	}
	current := getCurrent(next)
	updated, ok := safeAddInt64(*current, *delta)
	if !ok {
		return "invalid_quota_meter_overflow", map[string]any{"meter": meter}, true
	}
	*current = updated
	return "", nil, false
}
