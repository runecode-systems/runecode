package brokerapi

import "sync"

type runGatePlanCache struct {
	mu    sync.RWMutex
	byRun map[string]compiledRunGatePlan
}

func newRunGatePlanCache() *runGatePlanCache {
	return &runGatePlanCache{byRun: map[string]compiledRunGatePlan{}}
}

func (c *runGatePlanCache) get(runID string) (compiledRunGatePlan, bool) {
	if c == nil {
		return compiledRunGatePlan{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	plan, ok := c.byRun[runID]
	if !ok {
		return compiledRunGatePlan{}, false
	}
	return plan, true
}

func (c *runGatePlanCache) setPlan(runID string, plan compiledRunGatePlan) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byRun[runID] = plan
}

func (c *runGatePlanCache) invalidateRun(runID string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.byRun, runID)
}

func (c *runGatePlanCache) invalidateAll() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byRun = map[string]compiledRunGatePlan{}
}
