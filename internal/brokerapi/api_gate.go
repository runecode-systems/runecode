package brokerapi

import (
	"errors"
	"fmt"
	"sync"
)

type inFlightGate struct {
	mu             sync.Mutex
	limits         Limits
	perClientCount map[string]int
	perLaneCount   map[string]int
}

var errInFlightLimitExceeded = errors.New("in-flight limit exceeded")

func newInFlightGate(limits Limits) *inFlightGate {
	return &inFlightGate{
		limits:         limits,
		perClientCount: map[string]int{},
		perLaneCount:   map[string]int{},
	}
}

func (g *inFlightGate) acquire(clientID string, laneID string) (func(), error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if clientID == "" {
		clientID = "default-client"
	}
	if laneID == "" {
		laneID = "default-lane"
	}
	if g.perClientCount[clientID] >= g.limits.MaxInFlightPerClient {
		return nil, fmt.Errorf("%w: client %q has %d active, max %d", errInFlightLimitExceeded, clientID, g.perClientCount[clientID], g.limits.MaxInFlightPerClient)
	}
	if g.perLaneCount[laneID] >= g.limits.MaxInFlightPerLane {
		return nil, fmt.Errorf("%w: lane %q has %d active, max %d", errInFlightLimitExceeded, laneID, g.perLaneCount[laneID], g.limits.MaxInFlightPerLane)
	}
	g.perClientCount[clientID]++
	g.perLaneCount[laneID]++
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
	}, nil
}
