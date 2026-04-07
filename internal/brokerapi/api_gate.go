package brokerapi

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type inFlightGate struct {
	mu             sync.Mutex
	limits         Limits
	perClientCount map[string]int
	perLaneCount   map[string]int
	perClientRate  map[string]rateWindow
}

type rateWindow struct {
	windowStart int64
	count       int
}

var errInFlightLimitExceeded = errors.New("in-flight limit exceeded")
var errRateLimitExceeded = errors.New("rate limit exceeded")

func newInFlightGate(limits Limits) *inFlightGate {
	return &inFlightGate{
		limits:         limits,
		perClientCount: map[string]int{},
		perLaneCount:   map[string]int{},
		perClientRate:  map[string]rateWindow{},
	}
}

func (g *inFlightGate) acquire(clientID string, laneID string) (func(), error) {
	return g.acquireAt(clientID, laneID, time.Now().UTC())
}

func (g *inFlightGate) acquireAt(clientID string, laneID string, now time.Time) (func(), error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	clientID, laneID = normalizeGateIdentity(clientID, laneID)
	if err := g.enforceRateLimit(clientID, now); err != nil {
		return nil, err
	}
	if err := g.enforceInFlightLimits(clientID, laneID); err != nil {
		return nil, err
	}
	g.perClientCount[clientID]++
	g.perLaneCount[laneID]++
	return g.releaseFn(clientID, laneID), nil
}

func normalizeGateIdentity(clientID, laneID string) (string, string) {
	if clientID == "" {
		clientID = "default-client"
	}
	if laneID == "" {
		laneID = "default-lane"
	}
	return clientID, laneID
}

func (g *inFlightGate) enforceRateLimit(clientID string, now time.Time) error {
	if g.limits.MaxRequestsPerClientPS <= 0 {
		return nil
	}
	second := now.Unix()
	window := g.perClientRate[clientID]
	if window.windowStart != second {
		window.windowStart = second
		window.count = 0
	}
	if window.count >= g.limits.MaxRequestsPerClientPS {
		return fmt.Errorf("%w: client %q has %d requests in second %d, max %d", errRateLimitExceeded, clientID, window.count, second, g.limits.MaxRequestsPerClientPS)
	}
	window.count++
	g.perClientRate[clientID] = window
	return nil
}

func (g *inFlightGate) enforceInFlightLimits(clientID, laneID string) error {
	if g.perClientCount[clientID] >= g.limits.MaxInFlightPerClient {
		return fmt.Errorf("%w: client %q has %d active, max %d", errInFlightLimitExceeded, clientID, g.perClientCount[clientID], g.limits.MaxInFlightPerClient)
	}
	if g.perLaneCount[laneID] >= g.limits.MaxInFlightPerLane {
		return fmt.Errorf("%w: lane %q has %d active, max %d", errInFlightLimitExceeded, laneID, g.perLaneCount[laneID], g.limits.MaxInFlightPerLane)
	}
	return nil
}

func (g *inFlightGate) releaseFn(clientID, laneID string) func() {
	released := false
	return func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		if released {
			return
		}
		released = true
		g.perClientCount[clientID]--
		if g.perClientCount[clientID] <= 0 {
			delete(g.perClientCount, clientID)
		}
		g.perLaneCount[laneID]--
		if g.perLaneCount[laneID] <= 0 {
			delete(g.perLaneCount, laneID)
		}
	}
}
