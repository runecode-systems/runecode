package brokerapi

import "sync"

type compileCoordinator struct {
	sem      chan struct{}
	mu       sync.Mutex
	inFlight map[string]*compileInFlight
}

type compileInFlight struct {
	ready chan struct{}
	res   CompileAndPersistRunPlanResult
	err   error
}

func newCompileCoordinator(maxParallel int) *compileCoordinator {
	return &compileCoordinator{sem: make(chan struct{}, maxParallel), inFlight: map[string]*compileInFlight{}}
}

func (c *compileCoordinator) acquire() func() {
	c.sem <- struct{}{}
	return func() { <-c.sem }
}

func (c *compileCoordinator) startOrJoin(identity string) (*compileInFlight, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.inFlight[identity]; ok {
		return existing, false
	}
	created := &compileInFlight{ready: make(chan struct{})}
	c.inFlight[identity] = created
	return created, true
}

func (c *compileCoordinator) complete(identity string, call *compileInFlight, res CompileAndPersistRunPlanResult, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	call.res = res
	call.err = err
	close(call.ready)
	delete(c.inFlight, identity)
}
